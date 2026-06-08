package portscan

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

var ErrNoFreePort = errors.New("no free port found")

func FindFree(ctx context.Context, entries []Entry, opts FreeOptions) (int, error) {
	if opts.Address == "" {
		opts.Address = "127.0.0.1"
	}
	if !opts.UseTCP && !opts.UseUDP {
		opts.UseTCP = true
	}
	if opts.Start == nil || opts.End == nil {
		return 0, fmt.Errorf("free range is required")
	}
	used := make(map[int]map[string]bool)
	for _, e := range entries {
		if e.Port < *opts.Start || e.Port > *opts.End {
			continue
		}
		if used[e.Port] == nil {
			used[e.Port] = make(map[string]bool)
		}
		used[e.Port][e.Protocol] = true
	}
	for p := *opts.Start; p <= *opts.End; p++ {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
		if opts.UseTCP && used[p]["tcp"] {
			continue
		}
		if opts.UseUDP && used[p]["udp"] {
			continue
		}
		if bindAvailable(opts.Address, p, opts.UseTCP, opts.UseUDP) {
			return p, nil
		}
	}
	return 0, ErrNoFreePort
}

func bindAvailable(address string, port int, tcp, udp bool) bool {
	timeout := 2 * time.Second
	if tcp {
		lc := net.ListenConfig{}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		ln, err := lc.Listen(ctx, "tcp", net.JoinHostPort(address, fmt.Sprintf("%d", port)))
		cancel()
		if err != nil {
			return false
		}
		_ = ln.Close()
	}
	if udp {
		lc := net.ListenConfig{}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		pc, err := lc.ListenPacket(ctx, "udp", net.JoinHostPort(address, fmt.Sprintf("%d", port)))
		cancel()
		if err != nil {
			return false
		}
		_ = pc.Close()
	}
	return true
}
