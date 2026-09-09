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

func TestExecuteVersion(t *testing.T) {
	var out bytes.Buffer
	code := execute([]string{"--version"}, &out, ioDiscard(), func(tui.Model) error { return nil })
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(out.String(), version) {
		t.Fatalf("out = %q", out.String())
	}
}

func TestExecuteUsage(t *testing.T) {
	var errBuf bytes.Buffer
	code := execute([]string{"a.http", "b.http"}, ioDiscard(), &errBuf, func(tui.Model) error { return nil })
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(strings.ToLower(errBuf.String()), "accepts at most 1 arg") {
		t.Fatalf("err = %q", errBuf.String())
	}
}

func TestExecuteDefaultsToCurrentDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("api.http", []byte("GET https://example.com/ping\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var started bool
	code := execute([]string{}, ioDiscard(), ioDiscard(), func(tui.Model) error {
		started = true
		return nil
	})
	if code != 0 || !started {
		t.Fatalf("code=%d started=%v", code, started)
	}
}

func TestExecuteMissingFile(t *testing.T) {
	var errBuf bytes.Buffer
	code := execute([]string{filepath.Join(t.TempDir(), "nope.http")}, ioDiscard(), &errBuf, func(tui.Model) error { return nil })
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
}

func TestExecuteParsesAndStarts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api.http")
	if err := os.WriteFile(path, []byte("GET https://example.com/ping\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var started bool
	code := execute([]string{"--timeout", "1s", path}, ioDiscard(), ioDiscard(), func(m tui.Model) error {
		started = true
		return nil
	})
	if code != 0 || !started {
		t.Fatalf("code=%d started=%v", code, started)
	}
}

func TestExecuteOpensDirectory(t *testing.T) {
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
	code := execute([]string{dir}, ioDiscard(), ioDiscard(), func(tui.Model) error {
		started = true
		return nil
	})
	if code != 0 || !started {
		t.Fatalf("code=%d started=%v", code, started)
	}
}

func TestExecuteEmptyDirectory(t *testing.T) {
	var errBuf bytes.Buffer
	code := execute([]string{t.TempDir()}, ioDiscard(), &errBuf, func(tui.Model) error { return nil })
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
}

func TestExecuteStartError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api.http")
	if err := os.WriteFile(path, []byte("GET https://example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var errBuf bytes.Buffer
	code := execute([]string{path}, ioDiscard(), &errBuf, func(tui.Model) error {
		return errors.New("boom")
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(strings.ToLower(errBuf.String()), "boom") {
		t.Fatalf("err = %q", errBuf.String())
	}
}

func TestExecuteBadFlag(t *testing.T) {
	var errBuf bytes.Buffer
	code := execute([]string{"--nope"}, ioDiscard(), &errBuf, func(tui.Model) error { return nil })
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(strings.ToLower(errBuf.String()), "unknown flag") {
		t.Fatalf("err = %q", errBuf.String())
	}
}

func ioDiscard() *bytes.Buffer {
	return &bytes.Buffer{}
}
