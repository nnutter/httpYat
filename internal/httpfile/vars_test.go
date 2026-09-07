package httpfile

import (
	"testing"
)

func TestPrepareSubstitutesAndPrefixesHost(t *testing.T) {
	doc := parseTestdata(t, "host.http")
	req := doc.Requests[0]
	values := Resolve(doc, req, nil)
	got := Prepare(req, values)
	if got.URL != "https://httpbin.org/post" {
		t.Errorf("url = %q", got.URL)
	}
	if got.Method != "GET" {
		t.Errorf("method = %q", got.Method)
	}
}

func TestPrepareNestedVariablesAndOverlay(t *testing.T) {
	doc := parseTestdata(t, "variables.http")
	req := doc.Requests[0]
	values := Resolve(doc, req, map[string]string{"token": "secret"})
	got := Prepare(req, values)
	if got.URL != "https://example.com/anything?q=bar_Extended" {
		t.Errorf("url = %q", got.URL)
	}
	if len(got.Headers) != 1 || got.Headers[0].Value != "Bearer secret" {
		t.Errorf("headers = %+v", got.Headers)
	}
}

func TestPrepareLeavesUnknownPlaceholders(t *testing.T) {
	req := Request{
		Method: "GET",
		URL:    "https://example.com/{{missing}}",
		Headers: []Header{
			{Name: "X-Id", Value: "{{id}}"},
		},
		Body: `{"n":"{{name}}"}`,
	}
	got := Prepare(req, map[string]string{"id": "9"})
	if got.URL != "https://example.com/{{missing}}" {
		t.Errorf("url = %q", got.URL)
	}
	if got.Headers[0].Value != "9" {
		t.Errorf("header = %q", got.Headers[0].Value)
	}
	if got.Body != `{"n":"{{name}}"}` {
		t.Errorf("body = %q", got.Body)
	}
}

func TestPrepareUnescapesMustaches(t *testing.T) {
	req := Request{Method: "POST", URL: "https://example.com", Body: `\{\{literal\}\}`}
	got := Prepare(req, nil)
	if got.Body != "{{literal}}" {
		t.Errorf("body = %q", got.Body)
	}
}

func TestPrepareRequestVarsOverrideGlobals(t *testing.T) {
	doc := Document{
		Globals: []Variable{{Name: "id", Value: "global"}},
		Requests: []Request{{
			Method: "GET",
			URL:    "/{{id}}",
			Vars:   []Variable{{Name: "id", Value: "local"}},
		}},
	}
	req := doc.Requests[0]
	values := Resolve(doc, req, map[string]string{"id": "overlay"})
	got := Prepare(req, values)
	if got.URL != "/overlay" {
		t.Errorf("url = %q", got.URL)
	}
}

func TestDisplayVarsIncludesReferencedAndDefined(t *testing.T) {
	doc := parseTestdata(t, "multiple.http")
	req := doc.Requests[1]
	vars := DisplayVars(doc, req, map[string]string{"name": "Ada"})
	got := map[string]string{}
	for _, v := range vars {
		got[v.Name] = v.Value
	}
	if got["host"] != "https://api.example.com" {
		t.Errorf("host = %q", got["host"])
	}
	if got["name"] != "Ada" {
		t.Errorf("name = %q", got["name"])
	}
}

func TestReferencedExtractsSimpleIdentifiers(t *testing.T) {
	req := Request{
		URL: "{{host}}/users/{{id}}",
		Headers: []Header{
			{Value: "Bearer {{token}}"},
		},
		Body: `{{foo.bar}} and {{name}}`,
	}
	got := Referenced(req)
	want := []string{"host", "id", "token", "name"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
			return
		}
	}
}

func TestPrepareEmptyMethodDefaultsToGET(t *testing.T) {
	got := Prepare(Request{URL: "https://example.com"}, nil)
	if got.Method != "GET" {
		t.Errorf("method = %q", got.Method)
	}
}
