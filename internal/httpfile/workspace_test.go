package httpfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPathSingleFile(t *testing.T) {
	docs, err := LoadPath(filepath.Join("testdata", "simple.http"))
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || len(docs[0].Requests) != 1 {
		t.Fatalf("docs = %+v", docs)
	}
}

func TestLoadPathMissing(t *testing.T) {
	if _, err := LoadPath(filepath.Join(t.TempDir(), "nope.http")); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadDirCollectsSortedHTTPAndRest(t *testing.T) {
	dir := t.TempDir()
	writeHTTP(t, filepath.Join(dir, "b.http"), "GET https://example.com/b\n")
	writeHTTP(t, filepath.Join(dir, "a.rest"), "GET https://example.com/a\n")
	writeHTTP(t, filepath.Join(dir, "ignore.txt"), "GET https://example.com/skip\n")
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeHTTP(t, filepath.Join(dir, "sub", "nested.http"), "GET https://example.com/nested\n")

	docs, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("docs = %d, want 2", len(docs))
	}
	if got := filepath.Base(docs[0].Path); got != "a.rest" {
		t.Errorf("first = %q, want a.rest", got)
	}
	if got := filepath.Base(docs[1].Path); got != "b.http" {
		t.Errorf("second = %q, want b.http", got)
	}

	viaPath, err := LoadPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(viaPath) != 2 {
		t.Fatalf("via path docs = %d", len(viaPath))
	}
}

func TestLoadDirEmpty(t *testing.T) {
	if _, err := LoadDir(t.TempDir()); err == nil {
		t.Fatal("expected error for empty dir")
	}
}

func TestLoadDirParseError(t *testing.T) {
	dir := t.TempDir()
	writeHTTP(t, filepath.Join(dir, "bad.http"), "POST https://example.com/x\n\n< missing.json\n")
	if _, err := LoadDir(dir); err == nil {
		t.Fatal("expected parse error")
	}
}

func writeHTTP(t *testing.T, path, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
}
