package client

import (
	"bytes"
	"cmp"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const maxBody = 8 << 20

// Result is the outcome of sending a request through httpyac.
type Result struct {
	Status      string
	StatusCode  int
	Proto       string
	Headers     http.Header
	Body        []byte
	ContentType string
	Duration    time.Duration
	Truncated   bool
}

type sendJSON struct {
	Requests []sendRequest `json:"requests"`
}

type sendRequest struct {
	Duration float64       `json:"duration"`
	Response *sendResponse `json:"response"`
}

type sendResponse struct {
	Protocol      string         `json:"protocol"`
	HTTPVersion   string         `json:"httpVersion"`
	StatusCode    int            `json:"statusCode"`
	StatusMessage string         `json:"statusMessage"`
	Headers       map[string]any `json:"headers"`
	Body          any            `json:"body"`
}

func parseOutput(stdout []byte) (Result, error) {
	raw := bytes.TrimSpace(stdout)
	if len(raw) == 0 {
		return Result{}, errors.New("httpyac returned no JSON output")
	}
	if i := bytes.IndexByte(raw, '{'); i > 0 {
		raw = raw[i:]
	}

	var out sendJSON
	if err := json.Unmarshal(raw, &out); err != nil {
		return Result{}, fmt.Errorf("httpyac JSON: %w", err)
	}
	if len(out.Requests) == 0 {
		return Result{}, errors.New("httpyac did not run a matching request")
	}
	item := out.Requests[0]
	if item.Response == nil {
		return Result{}, errors.New("httpyac response is empty")
	}

	body := bodyBytes(item.Response.Body)
	truncated := len(body) > maxBody
	if truncated {
		body = body[:maxBody]
	}

	headers := asHeader(item.Response.Headers)
	proto := cmp.Or(item.Response.Protocol, item.Response.HTTPVersion)
	status := http.StatusText(item.Response.StatusCode)
	if item.Response.StatusMessage != "" {
		status = item.Response.StatusMessage
	}
	if item.Response.StatusCode > 0 {
		status = fmt.Sprintf("%d %s", item.Response.StatusCode, status)
	}

	return Result{
		Status:      status,
		StatusCode:  item.Response.StatusCode,
		Proto:       proto,
		Headers:     headers,
		Body:        body,
		ContentType: headers.Get("Content-Type"),
		Duration:    time.Duration(item.Duration) * time.Millisecond,
		Truncated:   truncated,
	}, nil
}

func asHeader(src map[string]any) http.Header {
	h := make(http.Header, len(src))
	for name, value := range src {
		switch v := value.(type) {
		case string:
			h.Add(name, v)
		case []any:
			for _, item := range v {
				h.Add(name, fmt.Sprint(item))
			}
		case nil:
		default:
			h.Add(name, fmt.Sprint(v))
		}
	}
	return h
}

func bodyBytes(v any) []byte {
	switch t := v.(type) {
	case nil:
		return make([]byte, 0)
	case string:
		return []byte(t)
	case map[string]any:
		if typ, _ := t["type"].(string); typ == "Buffer" {
			if data, ok := t["data"].([]any); ok {
				out := make([]byte, 0, len(data))
				for _, n := range data {
					f, ok := n.(float64)
					if !ok {
						continue
					}
					out = append(out, byte(f))
				}
				return out
			}
		}
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Appendf(nil, "%v", t)
		}
		return b
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Appendf(nil, "%v", t)
		}
		return b
	}
}
