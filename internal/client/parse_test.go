package client

import (
	"net/http"
	"strings"
	"testing"
)

func TestParseOutputResponse(t *testing.T) {
	src := `{
  "_meta": {"version": "1.1.0"},
  "requests": [{
    "name": "listUsers",
    "line": 3,
    "duration": 12,
    "response": {
      "protocol": "HTTP/1.1",
      "statusCode": 201,
      "statusMessage": "Created",
      "headers": {
        "content-type": "application/json",
        "x-trace": ["abc", "def"]
      },
      "body": {"id": 1, "ok": true}
    }
  }]
}`
	res, err := parseOutput([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Errorf("status = %d", res.StatusCode)
	}
	if !strings.Contains(res.Status, "201") {
		t.Errorf("status text = %q", res.Status)
	}
	if res.ContentType != "application/json" {
		t.Errorf("ct = %q", res.ContentType)
	}
	if got := res.Headers.Values("X-Trace"); strings.Join(got, ",") != "abc,def" {
		t.Errorf("headers = %v", res.Headers)
	}
	if !strings.Contains(string(res.Body), `"ok"`) {
		t.Errorf("body = %q", res.Body)
	}
	if res.Duration.Milliseconds() != 12 {
		t.Errorf("duration = %s", res.Duration)
	}
}

func TestParseOutputBufferBodyAndPreamble(t *testing.T) {
	src := "log line\n" + `{
  "requests": [{
    "response": {
      "statusCode": 200,
      "body": {"type": "Buffer", "data": [104, 105]}
    }
  }]
}`
	res, err := parseOutput([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if string(res.Body) != "hi" {
		t.Errorf("body = %q", res.Body)
	}
}

func TestParseOutputErrors(t *testing.T) {
	if _, err := parseOutput(nil); err == nil {
		t.Fatal("expected empty error")
	}
	if _, err := parseOutput([]byte(`{"requests":[]}`)); err == nil {
		t.Fatal("expected missing request")
	}
	if _, err := parseOutput([]byte(`not-json`)); err == nil {
		t.Fatal("expected json error")
	}
}
