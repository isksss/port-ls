package cli

import "testing"

func TestParseRangeArg(t *testing.T) {
	start, end, err := parseRangeArg("3000")
	if err != nil {
		t.Fatal(err)
	}
	if start != 3000 || end != 4000 {
		t.Fatalf("got %d-%d", start, end)
	}
	start, end, err = parseRangeArg("65000")
	if err != nil {
		t.Fatal(err)
	}
	if start != 65000 || end != 65535 {
		t.Fatalf("got %d-%d", start, end)
	}
	start, end, err = parseRangeArg("3000-3010")
	if err != nil {
		t.Fatal(err)
	}
	if start != 3000 || end != 3010 {
		t.Fatalf("got %d-%d", start, end)
	}
}

func TestNormalizeStateInput(t *testing.T) {
	got, err := normalizeStateInput("TIME-WAIT")
	if err != nil {
		t.Fatal(err)
	}
	if got != "time_wait" {
		t.Fatalf("got %q", got)
	}
}
