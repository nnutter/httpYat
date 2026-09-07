package httpfile

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// LoadPath loads one file or every .http/.rest file in a directory.
func LoadPath(path string) ([]Document, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		doc, err := ParseFile(path)
		if err != nil {
			return nil, err
		}
		return []Document{doc}, nil
	}
	return LoadDir(path)
}

// LoadDir loads the .http and .rest files directly inside dir, sorted.
func LoadDir(dir string) ([]Document, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".http", ".rest":
			names = append(names, e.Name())
		}
	}
	slices.Sort(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("no .http or .rest files in %s", dir)
	}
	docs := make([]Document, 0, len(names))
	for _, name := range names {
		doc, err := ParseFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}
