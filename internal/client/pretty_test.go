package client

import (
	"strings"
	"testing"
)

func TestBodyPrettyPrintsJSON(t *testing.T) {
	got := Body([]byte(`{"a":1,"b":[true,null,"x"]}`), "application/json", false)
	if !strings.Contains(got, "\n") || !strings.Contains(got, `"a"`) {
		t.Fatalf("not pretty: %q", got)
	}
	if strings.Contains(got, "\x1b") {
		t.Fatal("unexpected color")
	}
}

func TestBodyDetectsJSONWithoutContentType(t *testing.T) {
	got := Body([]byte(`[{"n": 2}]`), "", false)
	if !strings.Contains(got, `"n"`) {
		t.Fatalf("got %q", got)
	}
}

func TestBodyLeavesPlainText(t *testing.T) {
	got := Body([]byte("hello"), "text/plain", false)
	if got != "hello" {
		t.Fatalf("got %q", got)
	}
}

func TestBodyInvalidJSONWithJSONContentType(t *testing.T) {
	got := Body([]byte("not-json"), "application/json; charset=utf-8", false)
	if got != "not-json" {
		t.Fatalf("got %q", got)
	}
}

func TestBodyColorizesJSON(t *testing.T) {
	got := Body([]byte(`{"ok":true,"n":1,"s":"hi","z":null}`), "application/json", true)
	if !strings.Contains(got, "\x1b") {
		t.Fatalf("expected ANSI color, got %q", got)
	}
	plain := stripANSI(got)
	if !strings.Contains(plain, `"ok"`) || !strings.Contains(plain, "true") {
		t.Fatalf("lost text: %q", plain)
	}
}

func TestColorizeJSONHandlesEscapesAndNumbers(t *testing.T) {
	src := "{\n  \"k\\\"ey\": -2.5e+3\n}"
	got := stripANSI(colorizeJSON(src))
	if got != src {
		t.Fatalf("got %q", got)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			if i < len(s) {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
