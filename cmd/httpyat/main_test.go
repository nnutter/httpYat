package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nnutter/httpYat/internal/tui"
)

func TestRunVersion(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"httpyat", "-version"}, &out, ioDiscard(), func(tui.Model) error { return nil })
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(out.String(), version) {
		t.Fatalf("out = %q", out.String())
	}
}

func TestRunUsage(t *testing.T) {
	var err bytes.Buffer
	code := run([]string{"httpyat"}, ioDiscard(), &err, func(tui.Model) error { return nil })
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(err.String(), "usage:") {
		t.Fatalf("err = %q", err.String())
	}
}

func TestRunMissingFile(t *testing.T) {
	var err bytes.Buffer
	code := run([]string{"httpyat", filepath.Join(t.TempDir(), "nope.http")}, ioDiscard(), &err, func(tui.Model) error { return nil })
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
}

func TestRunParsesAndStarts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api.http")
	if err := os.WriteFile(path, []byte("GET https://example.com/ping\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var started bool
	code := run([]string{"httpyat", "-timeout", "1s", path}, ioDiscard(), ioDiscard(), func(m tui.Model) error {
		started = true
		return nil
	})
	if code != 0 || !started {
		t.Fatalf("code=%d started=%v", code, started)
	}
}

func TestRunOpensDirectory(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.http", "b.rest"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("GET https://example.com/x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	var started bool
	code := run([]string{"httpyat", dir}, ioDiscard(), ioDiscard(), func(tui.Model) error {
		started = true
		return nil
	})
	if code != 0 || !started {
		t.Fatalf("code=%d started=%v", code, started)
	}
}

func TestRunEmptyDirectory(t *testing.T) {
	var errBuf bytes.Buffer
	code := run([]string{"httpyat", t.TempDir()}, ioDiscard(), &errBuf, func(tui.Model) error { return nil })
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
}

func TestRunStartError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api.http")
	if err := os.WriteFile(path, []byte("GET https://example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var errBuf bytes.Buffer
	code := run([]string{"httpyat", path}, ioDiscard(), &errBuf, func(tui.Model) error {
		return errors.New("boom")
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(errBuf.String(), "boom") {
		t.Fatalf("err = %q", errBuf.String())
	}
}

func TestRunBadFlag(t *testing.T) {
	code := run([]string{"httpyat", "-nope"}, ioDiscard(), ioDiscard(), func(tui.Model) error { return nil })
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
}

func ioDiscard() *bytes.Buffer {
	return &bytes.Buffer{}
}
