package portscan

import (
	"sort"
	"strings"
)

func ApplyFilters(entries []Entry, opts ListOptions) []Entry {
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		e.Protocol = NormalizeProtocol(e.Protocol)
		e.State = NormalizeState(e.State)
		e.Address = NormalizeAddress(e.Address)
		if e.Name == "" {
			e.Name = "unknown"
		}
		if opts.Namespace != "" && e.Namespace == "" {
			e.Namespace = opts.Namespace
		}
		if e.Namespace == "" {
			e.Namespace = "local"
		}
		if !opts.All && !isDefaultVisibleState(e) {
			continue
		}
		if opts.Port != nil && e.Port != *opts.Port {
			continue
		}
		if !protocolAllowed(e.Protocol, opts.TCP, opts.UDP) {
			continue
		}
		if opts.Name != "" && !nameMatches(e.Name, opts.Name) {
			continue
		}
		if opts.Address != "" && !strings.Contains(strings.ToLower(e.Address), strings.ToLower(opts.Address)) {
			continue
		}
		if len(opts.States) > 0 && !stateAllowed(e.State, opts.States) {
			continue
		}
		out = append(out, e)
	}
	Sort(out)
	return out
}

func Sort(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Port != b.Port {
			return a.Port < b.Port
		}
		if a.Protocol != b.Protocol {
			return a.Protocol < b.Protocol
		}
		if a.Address != b.Address {
			return a.Address < b.Address
		}
		if a.State != b.State {
			return a.State < b.State
		}
		ap, bp := pidSort(a.PID), pidSort(b.PID)
		if ap != bp {
			return ap < bp
		}
		return a.Name < b.Name
	})
}

func isDefaultVisibleState(e Entry) bool {
	if e.State == "listen" {
		return true
	}
	return e.Protocol == "udp" && (e.State == "unknown" || e.State == "unconn")
}

func protocolAllowed(protocol string, tcp, udp bool) bool {
	if !tcp && !udp {
		return protocol == "tcp" || protocol == "udp"
	}
	return (tcp && protocol == "tcp") || (udp && protocol == "udp")
}

func stateAllowed(state string, states []string) bool {
	for _, v := range states {
		if state == NormalizeState(v) {
			return true
		}
	}
	return false
}

func nameMatches(name, query string) bool {
	if name == "" || name == "unknown" {
		return false
	}
	n := strings.ToLower(name)
	q := strings.ToLower(query)
	if strings.HasSuffix(n, ".exe") && !strings.HasSuffix(q, ".exe") {
		n = strings.TrimSuffix(n, ".exe")
	}
	if strings.HasSuffix(q, ".exe") && !strings.HasSuffix(n, ".exe") {
		q = strings.TrimSuffix(q, ".exe")
	}
	return strings.Contains(n, q)
}

func pidSort(pid *int) int {
	if pid == nil {
		return 1<<31 - 1
	}
	return *pid
}
