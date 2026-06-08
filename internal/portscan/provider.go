package portscan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var ErrNoProvider = errors.New("no port provider succeeded")

type commandProvider struct {
	name      string
	command   string
	args      func(Query) []string
	parse     func([]byte, string) ([]Entry, error)
	namespace string
}

func (p commandProvider) Name() string { return p.name }

func (p commandProvider) List(q Query) ([]Entry, []Diagnostic, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	args := p.args(q)
	cmd := exec.CommandContext(ctx, p.command, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	diag := Diagnostic{
		Provider:   p.name,
		Command:    p.command,
		ExitStatus: exitStatus(err),
		Stderr:     summarize(stderr.String()),
	}
	if err != nil {
		return nil, []Diagnostic{diag}, err
	}
	entries, err := p.parse(out, p.namespace)
	if err != nil {
		return nil, []Diagnostic{diag}, err
	}
	return entries, []Diagnostic{diag}, nil
}

type fallbackSet struct {
	providers []Provider
}

func (s fallbackSet) List(q Query) ([]Entry, []Diagnostic, error) {
	var diags []Diagnostic
	for _, p := range s.providers {
		entries, d, err := p.List(q)
		diags = append(diags, d...)
		if err == nil {
			return entries, diags, nil
		}
	}
	return nil, diags, ErrNoProvider
}

type combinedSet struct {
	sets []ProviderSet
}

func (s combinedSet) List(q Query) ([]Entry, []Diagnostic, error) {
	var all []Entry
	var diags []Diagnostic
	for _, set := range s.sets {
		entries, d, err := set.List(q)
		diags = append(diags, d...)
		if err != nil {
			return nil, diags, err
		}
		all = append(all, entries...)
	}
	return all, diags, nil
}

func NewDefaultProviderSet(includeHost bool) (ProviderSet, error) {
	local, err := localProviderSet()
	if err != nil {
		return nil, err
	}
	if !includeHost {
		return local, nil
	}
	if runtime.GOOS != "linux" || !IsWSL() {
		return nil, fmt.Errorf("--host is only supported on WSL")
	}
	host := fallbackSet{providers: []Provider{windowsPowerShellProvider("windows", "powershell.exe"), windowsNetstatProvider("windows", "netstat.exe")}}
	return combinedSet{sets: []ProviderSet{local, host}}, nil
}

func localProviderSet() (ProviderSet, error) {
	switch runtime.GOOS {
	case "linux":
		ns := "local"
		if IsWSL() {
			ns = "wsl"
		}
		return fallbackSet{providers: []Provider{
			linuxSSProvider(ns),
			linuxNetstatProvider(ns),
			lsofProvider(ns),
		}}, nil
	case "darwin":
		return fallbackSet{providers: []Provider{
			lsofProvider("local"),
			darwinNetstatProvider("local"),
		}}, nil
	case "windows":
		return fallbackSet{providers: []Provider{
			windowsPowerShellProvider("local", "powershell"),
			windowsNetstatProvider("local", "netstat"),
		}}, nil
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func IsWSL() bool {
	for _, path := range []string{"/proc/sys/kernel/osrelease", "/proc/version"} {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		v := strings.ToLower(string(b))
		if strings.Contains(v, "microsoft") || strings.Contains(v, "wsl") {
			return true
		}
	}
	return false
}

func exitStatus(err error) string {
	if err == nil {
		return "0"
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.String()
	}
	return err.Error()
}

func summarize(v string) string {
	v = strings.TrimSpace(v)
	if len(v) > 240 {
		return v[:240] + "..."
	}
	return v
}

func linuxSSProvider(namespace string) Provider {
	return commandProvider{
		name:      "linux:ss",
		command:   "ss",
		namespace: namespace,
		args: func(q Query) []string {
			if q.All {
				return []string{"-H", "-tuanpe"}
			}
			return []string{"-H", "-tulpen"}
		},
		parse: parseSS,
	}
}

func linuxNetstatProvider(namespace string) Provider {
	return commandProvider{
		name:      "linux:netstat",
		command:   "netstat",
		namespace: namespace,
		args: func(q Query) []string {
			if q.All {
				return []string{"-tunape"}
			}
			return []string{"-tunlpe"}
		},
		parse: parseNetstat,
	}
}

func darwinNetstatProvider(namespace string) Provider {
	return commandProvider{
		name:      "darwin:netstat",
		command:   "netstat",
		namespace: namespace,
		args:      func(Query) []string { return []string{"-anv"} },
		parse:     parseNetstat,
	}
}

func lsofProvider(namespace string) Provider {
	return commandProvider{
		name:      "lsof",
		command:   "lsof",
		namespace: namespace,
		args:      func(Query) []string { return []string{"-nP", "-iTCP", "-iUDP"} },
		parse:     parseLsof,
	}
}

func windowsPowerShellProvider(namespace, command string) Provider {
	script := "$ErrorActionPreference='Stop';" +
		"Get-NetTCPConnection | ForEach-Object { $p=$null; try{$p=(Get-Process -Id $_.OwningProcess -ErrorAction Stop).ProcessName}catch{}; \"tcp`t$($_.LocalAddress)`t$($_.LocalPort)`t$($_.State)`t$($_.OwningProcess)`t$p\" };" +
		"Get-NetUDPEndpoint | ForEach-Object { $p=$null; try{$p=(Get-Process -Id $_.OwningProcess -ErrorAction Stop).ProcessName}catch{}; \"udp`t$($_.LocalAddress)`t$($_.LocalPort)`tunknown`t$($_.OwningProcess)`t$p\" }"
	return commandProvider{
		name:      "windows:powershell",
		command:   command,
		namespace: namespace,
		args:      func(Query) []string { return []string{"-NoProfile", "-Command", script} },
		parse:     parseWindowsTSV,
	}
}

func windowsNetstatProvider(namespace, command string) Provider {
	return commandProvider{
		name:      "windows:netstat",
		command:   command,
		namespace: namespace,
		args:      func(Query) []string { return []string{"-ano"} },
		parse:     parseNetstat,
	}
}
