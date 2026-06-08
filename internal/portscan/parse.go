package portscan

import (
	"bufio"
	"bytes"
	"regexp"
	"strconv"
	"strings"
)

var ssProcessRE = regexp.MustCompile(`"([^"]+)".*pid=([0-9]+)`)

func parseSS(out []byte, namespace string) ([]Entry, error) {
	var entries []Entry
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		proto := NormalizeProtocol(fields[0])
		state := NormalizeState(fields[1])
		local := fields[4]
		host, port, ok := SplitAddressPort(local)
		if !ok && len(fields) > 5 {
			host, port, ok = SplitAddressPort(fields[5])
		}
		if !ok {
			continue
		}
		pid, name := parseSSProcess(line)
		entries = append(entries, Entry{
			Port: port, Protocol: proto, Address: NormalizeAddress(host), State: state,
			PID: pid, Name: unknownName(name), Namespace: namespace,
		})
	}
	return entries, sc.Err()
}

func parseSSProcess(line string) (*int, string) {
	m := ssProcessRE.FindStringSubmatch(line)
	if len(m) != 3 {
		return nil, "unknown"
	}
	p, err := strconv.Atoi(m[2])
	if err != nil {
		return nil, m[1]
	}
	return &p, m[1]
}

func parseNetstat(out []byte, namespace string) ([]Entry, error) {
	var entries []Entry
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		lower := strings.ToLower(line)
		if !strings.HasPrefix(lower, "tcp") && !strings.HasPrefix(lower, "udp") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		proto := NormalizeProtocol(fields[0])
		localIdx := netstatLocalIndex(fields)
		if len(fields) <= localIdx {
			continue
		}
		host, port, ok := SplitAddressPort(fields[localIdx])
		if !ok {
			continue
		}
		state := "unknown"
		if proto == "tcp" {
			for _, f := range fields {
				n := NormalizeState(f)
				if n == "listen" || n == "established" || strings.Contains(n, "wait") || strings.Contains(n, "syn") || strings.Contains(n, "close") {
					state = n
					break
				}
			}
		}
		pid, name := parseNetstatPIDName(fields)
		entries = append(entries, Entry{
			Port: port, Protocol: proto, Address: NormalizeAddress(host), State: state,
			PID: pid, Name: unknownName(name), Namespace: namespace,
		})
	}
	return entries, sc.Err()
}

func netstatLocalIndex(fields []string) int {
	if len(fields) > 3 && isNumeric(fields[1]) && isNumeric(fields[2]) {
		return 3
	}
	return 1
}

func isNumeric(v string) bool {
	_, err := strconv.Atoi(v)
	return err == nil
}

func parseNetstatPIDName(fields []string) (*int, string) {
	for i := len(fields) - 1; i >= 0; i-- {
		f := fields[i]
		if strings.Contains(f, "/") {
			parts := strings.SplitN(f, "/", 2)
			p, err := strconv.Atoi(parts[0])
			if err == nil {
				return &p, parts[1]
			}
		}
		if p, err := strconv.Atoi(f); err == nil && p > 0 {
			return &p, "unknown"
		}
	}
	return nil, "unknown"
}

func parseLsof(out []byte, namespace string) ([]Entry, error) {
	var entries []Entry
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "COMMAND") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		p, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		name := fields[0]
		proto := ""
		state := "unknown"
		local := ""
		for i, f := range fields {
			if strings.EqualFold(f, "TCP") || strings.EqualFold(f, "UDP") {
				proto = NormalizeProtocol(f)
				if i+1 < len(fields) {
					local = fields[i+1]
				}
				break
			}
		}
		if proto == "" {
			continue
		}
		if idx := strings.Index(line, "(LISTEN)"); idx >= 0 {
			_ = idx
			state = "listen"
		} else if idx := strings.Index(line, "("); idx >= 0 {
			if end := strings.Index(line[idx:], ")"); end > 0 {
				state = NormalizeState(line[idx+1 : idx+end])
			}
		}
		host, port, ok := SplitAddressPort(local)
		if !ok {
			continue
		}
		entries = append(entries, Entry{
			Port: port, Protocol: proto, Address: NormalizeAddress(host), State: state,
			PID: &p, Name: unknownName(name), Namespace: namespace,
		})
	}
	return entries, sc.Err()
}

func parseWindowsTSV(out []byte, namespace string) ([]Entry, error) {
	var entries []Entry
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		fields := strings.Split(sc.Text(), "\t")
		if len(fields) < 6 {
			continue
		}
		port, err := strconv.Atoi(strings.TrimSpace(fields[2]))
		if err != nil || !ValidPort(port) {
			continue
		}
		var pid *int
		if p, err := strconv.Atoi(strings.TrimSpace(fields[4])); err == nil && p > 0 {
			pid = &p
		}
		entries = append(entries, Entry{
			Port: port, Protocol: NormalizeProtocol(fields[0]), Address: NormalizeAddress(fields[1]),
			State: NormalizeState(fields[3]), PID: pid, Name: unknownName(fields[5]), Namespace: namespace,
		})
	}
	return entries, sc.Err()
}

func unknownName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "<nil>" {
		return "unknown"
	}
	return name
}
