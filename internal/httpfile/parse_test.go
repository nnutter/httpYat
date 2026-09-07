package httpfile

import (
	"path/filepath"
	"testing"
)

func TestParseFileSimpleGET(t *testing.T) {
	doc := parseTestdata(t, "simple.http")
	if len(doc.Requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(doc.Requests))
	}
	req := doc.Requests[0]
	if req.Method != "GET" {
		t.Errorf("method = %q", req.Method)
	}
	if req.URL != "https://example.com/users" {
		t.Errorf("url = %q", req.URL)
	}
	if len(req.Headers) != 1 || req.Headers[0].Name != "Accept" {
		t.Errorf("headers = %+v", req.Headers)
	}
}

func TestParseFileMultipleRequestsAndGlobals(t *testing.T) {
	doc := parseTestdata(t, "multiple.http")
	if len(doc.Globals) != 1 || doc.Globals[0].Name != "host" {
		t.Fatalf("globals = %+v", doc.Globals)
	}
	if len(doc.Requests) != 3 {
		t.Fatalf("requests = %d, want 3", len(doc.Requests))
	}
	if doc.Requests[0].Name != "listUsers" {
		t.Errorf("first name = %q", doc.Requests[0].Name)
	}
	if doc.Requests[1].Name != "createUser" {
		t.Errorf("second name = %q", doc.Requests[1].Name)
	}
	if doc.Requests[1].Description != "Register a user" {
		t.Errorf("description = %q", doc.Requests[1].Description)
	}
	if doc.Requests[1].Method != "POST" {
		t.Errorf("method = %q", doc.Requests[1].Method)
	}
	if want := "{\n  \"name\": \"{{name}}\"\n}"; doc.Requests[1].Body != want {
		t.Errorf("body = %q, want %q", doc.Requests[1].Body, want)
	}
	if doc.Requests[2].Name != "GET {{host}}/health" {
		t.Errorf("anonymous name = %q", doc.Requests[2].Name)
	}
}

func TestParseFileCommentsAndDescription(t *testing.T) {
	doc := parseTestdata(t, "comments.http")
	if len(doc.Requests) != 1 {
		t.Fatalf("requests = %d", len(doc.Requests))
	}
	if doc.Requests[0].Description != "first comment becomes description" {
		t.Errorf("description = %q", doc.Requests[0].Description)
	}
	if doc.Requests[0].URL != "{{host}}/thing" {
		t.Errorf("url = %q", doc.Requests[0].URL)
	}
}

func TestParseFileStoredResponseIgnored(t *testing.T) {
	doc := parseTestdata(t, "stored.http")
	if len(doc.Requests) != 1 {
		t.Fatalf("requests = %d", len(doc.Requests))
	}
	if doc.Requests[0].Body != "" {
		t.Errorf("body = %q, want empty", doc.Requests[0].Body)
	}
}

func TestParseFileMultilineURL(t *testing.T) {
	doc := parseTestdata(t, "multiline.http")
	if got := doc.Requests[0].URL; got != "https://example.com/search?q={{query}}&limit=10" {
		t.Errorf("url = %q", got)
	}
}

func TestParseFileIncludeBody(t *testing.T) {
	doc := parseTestdata(t, "include.http")
	if got := doc.Requests[0].Body; got != `{"id": 7, "ok": true}` {
		t.Errorf("body = %q", got)
	}
}

func TestParseDefaultGETAndRelativeURL(t *testing.T) {
	doc := mustParse(t, "https://example.com/plain\n\n###\n/status\n")
	if len(doc.Requests) != 2 {
		t.Fatalf("requests = %d", len(doc.Requests))
	}
	if doc.Requests[0].Method != "GET" || doc.Requests[0].URL != "https://example.com/plain" {
		t.Errorf("first = %+v", doc.Requests[0])
	}
	if doc.Requests[1].Method != "GET" || doc.Requests[1].URL != "/status" {
		t.Errorf("second = %+v", doc.Requests[1])
	}
}

func TestParseCRLFAndRequestVersion(t *testing.T) {
	doc := mustParse(t, "POST https://example.com/v1 HTTP/1.1\r\nContent-Type: application/json\r\n\r\n{\"a\":1}\r\n")
	req := doc.Requests[0]
	if req.Method != "POST" || req.Version != "HTTP/1.1" {
		t.Errorf("request = %+v", req)
	}
	if req.Body != `{"a":1}` {
		t.Errorf("body = %q", req.Body)
	}
}

func TestParseRequestScopedVarsAndTitle(t *testing.T) {
	src := "@global = 1\n### Shown Title\n@id := 42\nGET /items/{{id}}\n"
	doc := mustParse(t, src)
	if len(doc.Globals) != 1 || doc.Globals[0].Value != "1" {
		t.Fatalf("globals = %+v", doc.Globals)
	}
	req := doc.Requests[0]
	if req.Name != "Shown Title" {
		t.Errorf("name = %q", req.Name)
	}
	if len(req.Vars) != 1 || req.Vars[0].Name != "id" || req.Vars[0].Value != "42" {
		t.Errorf("vars = %+v", req.Vars)
	}
}

func TestParseMetadataNameWinsOverTitle(t *testing.T) {
	src := "### From Separator\n# @title From Title\n# @name fromName\nGET /x\n"
	doc := mustParse(t, src)
	if doc.Requests[0].Name != "fromName" {
		t.Errorf("name = %q", doc.Requests[0].Name)
	}
	if doc.Requests[0].MetaName() != "fromName" {
		t.Errorf("meta name = %q", doc.Requests[0].MetaName())
	}
}

func TestParseSeparatorTitleIsNotMetaName(t *testing.T) {
	doc := mustParse(t, "### Shown Title\nGET /x\n")
	if doc.Requests[0].Name != "Shown Title" {
		t.Errorf("name = %q", doc.Requests[0].Name)
	}
	if doc.Requests[0].MetaName() != "" {
		t.Errorf("meta name = %q", doc.Requests[0].MetaName())
	}
}

func TestParseSkipsScripts(t *testing.T) {
	src := "GET https://example.com/x\n\n{\"ok\": true}\n\n> {% client.log('hi') %}\n"
	doc := mustParse(t, src)
	if doc.Requests[0].Body != `{"ok": true}` {
		t.Errorf("body = %q", doc.Requests[0].Body)
	}
}

func TestParseFileMissing(t *testing.T) {
	_, err := ParseFile(filepath.Join(t.TempDir(), "missing.http"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseIncludeMissingFile(t *testing.T) {
	_, err := Parse("POST /x\n\n< no-such.json\n", t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseEmptyAndUnknownPreamble(t *testing.T) {
	doc := mustParse(t, "\n\nnot a request\n")
	if len(doc.Requests) != 0 {
		t.Fatalf("requests = %+v", doc.Requests)
	}
}

func parseTestdata(t *testing.T, name string) Document {
	t.Helper()
	doc, err := ParseFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func mustParse(t *testing.T, src string) Document {
	t.Helper()
	doc, err := Parse(src, "")
	if err != nil {
		t.Fatal(err)
	}
	return doc
}
