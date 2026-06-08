package portscan

import (
	"net"
	"regexp"
	"strconv"
	"strings"
)

var stateTokenRE = regexp.MustCompile(`[^a-z0-9]+`)

func NormalizeProtocol(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if strings.HasPrefix(v, "tcp") {
		return "tcp"
	}
	if strings.HasPrefix(v, "udp") {
		return "udp"
	}
	return v
}

func NormalizeState(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "-" {
		return "unknown"
	}
	v = strings.ToLower(v)
	v = strings.TrimPrefix(v, "state:")
	v = stateTokenRE.ReplaceAllString(v, "_")
	v = strings.Trim(v, "_")
	if v == "" {
		return "unknown"
	}
	if v == "listening" {
		return "listen"
	}
	return v
}

func NormalizeAddress(v string) string {
	host, _, ok := SplitAddressPort(v)
	if ok {
		v = host
	}
	v = strings.TrimSpace(v)
	v = strings.Trim(v, "[]")
	switch v {
	case "", "*", "0.0.0.0", "0.0.0.0%0":
		return "0.0.0.0"
	case "::", ":::", "[::]", "[::]%0":
		return "::"
	}
	if strings.HasPrefix(v, "*:") {
		return "0.0.0.0"
	}
	if strings.HasPrefix(v, ":::") {
		return "::"
	}
	return v
}

func SplitAddressPort(v string) (string, int, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", 0, false
	}
	if strings.HasPrefix(v, "[") {
		host, port, err := net.SplitHostPort(v)
		if err == nil {
			p, ok := parsePort(port)
			return strings.Trim(host, "[]"), p, ok
		}
	}
	if host, port, err := net.SplitHostPort(v); err == nil {
		p, ok := parsePort(port)
		return host, p, ok
	}
	idx := strings.LastIndex(v, ":")
	if idx < 0 {
		return v, 0, false
	}
	portPart := v[idx+1:]
	p, ok := parsePort(portPart)
	if !ok {
		return v, 0, false
	}
	host := v[:idx]
	host = strings.Trim(host, "[]")
	if host == "" || host == "*" {
		host = "0.0.0.0"
	}
	if strings.Count(host, ":") > 0 && strings.Trim(host, ":") == "" {
		host = "::"
	}
	return host, p, true
}

func parsePort(v string) (int, bool) {
	v = strings.TrimSpace(v)
	if v == "*" {
		return 0, false
	}
	p, err := strconv.Atoi(v)
	if err != nil || p < 1 || p > 65535 {
		return 0, false
	}
	return p, true
}

func ValidPort(p int) bool {
	return p >= 1 && p <= 65535
}
