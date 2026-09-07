package client

import (
	"slices"
	"testing"
	"time"
)

func TestInputArgsNamedRequest(t *testing.T) {
	in := Input{
		File: "api.http",
		Line: 12,
		Name: "listUsers",
		Vars: map[string]string{"id": "3", "host": "https://example.com"},
	}
	got := in.args(30 * time.Second)
	want := []string{
		"send", "--json", "--raw", "--output", "response", "--no-color",
		"--timeout", "30000",
		"-n", "listUsers",
		"api.http",
		"--var", "host=https://example.com",
		"--var", "id=3",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestInputArgsFilePrecedesVars(t *testing.T) {
	// httpyac declares --var as variadic, so it swallows any following
	// argument; the file must come first or it is eaten as a var value.
	in := Input{File: "api.http", Line: 1, Vars: map[string]string{"id": "3"}}
	got := in.args(0)
	file := slices.Index(got, "api.http")
	flag := slices.Index(got, "--var")
	if file < 0 || flag < 0 || file > flag {
		t.Fatalf("file must precede --var: %#v", got)
	}
}

func TestInputArgsLineIsZeroBased(t *testing.T) {
	in := Input{File: "a.http", Line: 1}
	got := in.args(0)
	if !slices.Contains(got, "-l") {
		t.Fatalf("missing -l: %#v", got)
	}
	i := slices.Index(got, "-l")
	if got[i+1] != "0" {
		t.Fatalf("line = %q, want 0", got[i+1])
	}
	if slices.Contains(got, "-n") {
		t.Fatalf("unexpected -n: %#v", got)
	}
}
