package httpfile

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

const (
	statePreamble = iota
	stateRequest
	stateHeaders
	stateBody
)

var methods = []string{
	"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS",
	"CONNECT", "TRACE", "PROPFIND", "PROPPATCH", "MKCOL",
	"COPY", "MOVE", "LOCK", "UNLOCK", "CHECKOUT", "CHECKIN",
	"REPORT", "MERGE", "MKACTIVITY", "MKWORKSPACE", "VERSION-CONTROL",
	"BASELINE-CONTROL",
}

type srcLine struct {
	Number int
	Text   string
}

// ParseFile reads and parses an httpYac request file.
func ParseFile(path string) (Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	doc, err := parse(string(data), filepath.Dir(path))
	if err != nil {
		return Document{}, err
	}
	doc.Path = path
	return doc, nil
}

// Parse parses request file contents. Includes are resolved relative to dir.
func Parse(src, dir string) (Document, error) {
	return parse(src, dir)
}

func parse(src, dir string) (Document, error) {
	src = strings.TrimPrefix(src, "\ufeff")
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	raw := strings.Split(src, "\n")
	lines := numbered(raw)
	lines = stripBlockComments(lines)

	var doc Document
	for _, region := range splitRegions(lines) {
		req, err := parseRegion(region.title, region.lines, dir)
		if err != nil {
			return Document{}, err
		}
		if !hasRequest(req) {
			doc.Globals = append(doc.Globals, req.Vars...)
			continue
		}
		doc.Requests = append(doc.Requests, req)
	}
	return doc, nil
}

func numbered(lines []string) []srcLine {
	out := make([]srcLine, len(lines))
	for i, line := range lines {
		out[i] = srcLine{Number: i + 1, Text: line}
	}
	return out
}

func stripBlockComments(lines []srcLine) []srcLine {
	out := make([]srcLine, 0, len(lines))
	inBlock := false
	for _, line := range lines {
		trim := strings.TrimSpace(line.Text)
		if inBlock {
			if blockCommentEnds(trim) {
				inBlock = false
			}
			continue
		}
		if blockCommentStarts(trim) {
			if !blockCommentEnds(trim) {
				inBlock = true
			}
			continue
		}
		out = append(out, line)
	}
	return out
}

type rawRegion struct {
	title string
	lines []srcLine
}

func splitRegions(lines []srcLine) []rawRegion {
	var regions []rawRegion
	current := rawRegion{}
	started := false
	for _, line := range lines {
		if !isSeparator(line.Text) {
			current.lines = append(current.lines, line)
			started = true
			continue
		}
		if started || len(current.lines) > 0 || current.title != "" {
			regions = append(regions, current)
		}
		current = rawRegion{title: separatorTitle(line.Text)}
		started = true
	}
	if started || len(current.lines) > 0 || current.title != "" {
		regions = append(regions, current)
	}
	return regions
}

func parseRegion(title string, lines []srcLine, dir string) (Request, error) {
	req := Request{Name: strings.TrimSpace(title)}
	state := statePreamble
	body := make([]srcLine, 0)
	sawRequest := false

	for _, line := range lines {
		trim := strings.TrimSpace(line.Text)
		if isBlank(trim) {
			if inHeadersOrRequest(state) {
				state = stateBody
			}
			if state == stateBody {
				body = append(body, line)
			}
			continue
		}
		if sawRequest && isStoredResponse(trim) {
			break
		}
		if sawRequest && isScript(trim) {
			break
		}
		switch state {
		case statePreamble:
			if handlePreamble(&req, line, trim) {
				continue
			}
			method, url, version, ok := parseRequestLine(trim)
			if !ok {
				continue
			}
			req.Method = method
			req.URL = url
			req.Version = version
			req.Line = line.Number
			state = stateRequest
			sawRequest = true
		case stateRequest:
			if isURLContinuation(line.Text) {
				req.URL += strings.TrimSpace(line.Text)
				continue
			}
			if name, value, ok := parseHeader(trim); ok {
				req.Headers = append(req.Headers, Header{Name: name, Value: value})
				state = stateHeaders
				continue
			}
			state = stateBody
			body = append(body, line)
		case stateHeaders:
			if name, value, ok := parseHeader(trim); ok {
				req.Headers = append(req.Headers, Header{Name: name, Value: value})
				continue
			}
			state = stateBody
			body = append(body, line)
		case stateBody:
			body = append(body, line)
		}
	}

	expanded, err := expandBody(joinBody(body), dir)
	if err != nil {
		return Request{}, err
	}
	req.Body = expanded
	if req.Name == "" {
		req.Name = req.DisplayName()
	}
	return req, nil
}

func handlePreamble(req *Request, line srcLine, trim string) bool {
	if isComment(trim) {
		if key, value, ok := parseMetadata(trim); ok {
			applyMetadata(req, key, value)
			return true
		}
		if req.Description == "" {
			req.Description = commentText(trim)
		}
		return true
	}
	if name, value, ok := parseVariable(trim); ok {
		req.Vars = append(req.Vars, Variable{Name: name, Value: value, Line: line.Number})
		return true
	}
	return false
}

func applyMetadata(req *Request, key, value string) {
	switch strings.ToLower(key) {
	case "name":
		req.Name = value
		req.explicitName = true
	case "title":
		if !req.explicitName {
			req.Name = value
		}
	case "description":
		req.Description = value
	}
}

func inHeadersOrRequest(state int) bool {
	return state == stateHeaders || state == stateRequest
}

func hasRequest(req Request) bool {
	return req.Method != "" || req.URL != ""
}

func isBlank(s string) bool {
	return s == ""
}

func isSeparator(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "###")
}

func separatorTitle(line string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "###"))
}

func blockCommentStarts(trim string) bool {
	return strings.HasPrefix(trim, "/*")
}

func blockCommentEnds(trim string) bool {
	return strings.Contains(trim, "*/")
}

func isComment(trim string) bool {
	return strings.HasPrefix(trim, "#") || strings.HasPrefix(trim, "//")
}

func commentText(trim string) string {
	switch {
	case strings.HasPrefix(trim, "//"):
		return strings.TrimSpace(strings.TrimPrefix(trim, "//"))
	case strings.HasPrefix(trim, "#"):
		return strings.TrimSpace(strings.TrimPrefix(trim, "#"))
	default:
		return trim
	}
}

func parseMetadata(trim string) (string, string, bool) {
	text := commentText(trim)
	if !strings.HasPrefix(text, "@") {
		return "", "", false
	}
	body := strings.TrimSpace(strings.TrimPrefix(text, "@"))
	key, value, found := strings.Cut(body, " ")
	if !found {
		return key, "", key != ""
	}
	return key, strings.TrimSpace(value), key != ""
}

func parseVariable(trim string) (string, string, bool) {
	if !strings.HasPrefix(trim, "@") {
		return "", "", false
	}
	body := strings.TrimSpace(strings.TrimPrefix(trim, "@"))
	name, rest, found := strings.Cut(body, ":=")
	if !found {
		name, rest, found = strings.Cut(body, "=")
	}
	if !found {
		return "", "", false
	}
	name = strings.TrimSpace(name)
	if !isIdent(name) {
		return "", "", false
	}
	return name, strings.TrimSpace(rest), true
}

func isIdent(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !isIdentStart(r) {
				return false
			}
			continue
		}
		if !isIdentPart(r) {
			return false
		}
	}
	return true
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return isIdentStart(r) || unicode.IsDigit(r)
}

func parseRequestLine(trim string) (method, url, version string, ok bool) {
	fields := strings.Fields(trim)
	if len(fields) == 0 {
		return "", "", "", false
	}
	if isHTTPVersion(fields[len(fields)-1]) {
		version = fields[len(fields)-1]
		fields = fields[:len(fields)-1]
	}
	if len(fields) == 0 {
		return "", "", "", false
	}
	if isMethod(fields[0]) {
		method = fields[0]
		if len(fields) < 2 {
			return "", "", "", false
		}
		return method, fields[1], version, true
	}
	if looksLikeURL(fields[0]) {
		return "GET", fields[0], version, true
	}
	return "", "", "", false
}

func isMethod(token string) bool {
	return slices.Contains(methods, token)
}

func looksLikeURL(token string) bool {
	return hasScheme(token) || isAbsolutePath(token) || startsWithVar(token)
}

func hasScheme(token string) bool {
	return strings.Contains(token, "://")
}

func isAbsolutePath(token string) bool {
	return strings.HasPrefix(token, "/")
}

func startsWithVar(token string) bool {
	return strings.HasPrefix(token, "{{")
}

func isHTTPVersion(token string) bool {
	return strings.HasPrefix(token, "HTTP/")
}

func isURLContinuation(line string) bool {
	if !isIndented(line) {
		return false
	}
	trim := strings.TrimSpace(line)
	return hasContinuationPrefix(trim)
}

func isIndented(line string) bool {
	return line != "" && (line[0] == ' ' || line[0] == '\t')
}

func hasContinuationPrefix(trim string) bool {
	return strings.HasPrefix(trim, "/") ||
		strings.HasPrefix(trim, "?") ||
		strings.HasPrefix(trim, "&") ||
		strings.HasPrefix(trim, "{{")
}

func parseHeader(trim string) (string, string, bool) {
	if looksLikeURL(strings.Fields(trim)[0]) {
		return "", "", false
	}
	name, value, found := strings.Cut(trim, ":")
	if !found {
		return "", "", false
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, " \t") {
		return "", "", false
	}
	return name, strings.TrimSpace(value), true
}

func isStoredResponse(trim string) bool {
	return strings.HasPrefix(trim, "HTTP/")
}

func isScript(trim string) bool {
	return strings.HasPrefix(trim, "> {%") || strings.HasPrefix(trim, ">{%")
}

func joinBody(lines []srcLine) string {
	if len(lines) == 0 {
		return ""
	}
	start := 0
	for start < len(lines) && isBlank(strings.TrimSpace(lines[start].Text)) {
		start++
	}
	end := len(lines)
	for end > start && isBlank(strings.TrimSpace(lines[end-1].Text)) {
		end--
	}
	if start >= end {
		return ""
	}
	parts := make([]string, 0, end-start)
	for _, line := range lines[start:end] {
		parts = append(parts, line.Text)
	}
	return strings.Join(parts, "\n")
}

func expandBody(body, dir string) (string, error) {
	if body == "" {
		return "", nil
	}
	lines := strings.Split(body, "\n")
	var b strings.Builder
	for i, line := range lines {
		if path, ok := includePath(strings.TrimSpace(line)); ok {
			data, err := os.ReadFile(filepath.Join(dir, path))
			if err != nil {
				return "", err
			}
			b.Write(bytesTrimRightNewline(data))
		} else {
			b.WriteString(line)
		}
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String(), nil
}

func includePath(trim string) (string, bool) {
	rest, ok := strings.CutPrefix(trim, "<")
	if !ok {
		return "", false
	}
	rest = strings.TrimSpace(rest)
	return rest, rest != ""
}

func bytesTrimRightNewline(data []byte) []byte {
	return []byte(strings.TrimRight(string(data), "\n\r"))
}
