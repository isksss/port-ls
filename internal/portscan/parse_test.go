package portscan

import "testing"

func TestParseSS(t *testing.T) {
	input := []byte(`tcp LISTEN 0 4096 127.0.0.1:3000 0.0.0.0:* users:(("node",pid=1234,fd=12))
udp UNCONN 0 0 0.0.0.0:5353 0.0.0.0:* users:(("avahi",pid=55,fd=3))
`)
	got, err := parseSS(input, "local")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Port != 3000 || got[0].Protocol != "tcp" || got[0].State != "listen" || *got[0].PID != 1234 || got[0].Name != "node" {
		t.Fatalf("unexpected tcp entry: %+v", got[0])
	}
	if got[1].Port != 5353 || got[1].Protocol != "udp" || got[1].Address != "0.0.0.0" {
		t.Fatalf("unexpected udp entry: %+v", got[1])
	}
}

func TestParseWindowsTSV(t *testing.T) {
	input := []byte("tcp\t127.0.0.1\t5432\tListen\t222\tpostgres\nudp\t::\t5353\tunknown\t0\t\n")
	got, err := parseWindowsTSV(input, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].State != "listen" || got[0].Name != "postgres" || got[0].Namespace != "windows" {
		t.Fatalf("unexpected entry: %+v", got[0])
	}
	if got[1].PID != nil || got[1].Name != "unknown" || got[1].Address != "::" {
		t.Fatalf("unexpected udp entry: %+v", got[1])
	}
}

func TestApplyFiltersAndSort(t *testing.T) {
	pid := 10
	entries := []Entry{
		{Port: 3001, Protocol: "TCP", Address: "127.0.0.1", State: "LISTENING", PID: &pid, Name: "Node.EXE"},
		{Port: 3000, Protocol: "udp", Address: "*", State: "", Name: ""},
	}
	port := 3001
	got := ApplyFilters(entries, ListOptions{Port: &port, Name: "node"})
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Protocol != "tcp" || got[0].State != "listen" || got[0].Namespace != "local" {
		t.Fatalf("unexpected normalized entry: %+v", got[0])
	}
}
