package httpfile

import (
	"maps"
	"regexp"
	"strings"
)

const expandPasses = 8

var refPattern = regexp.MustCompile(`\{\{([A-Za-z_][A-Za-z0-9_]*)\}\}`)

// Resolve builds the effective variable map for a request.
// Overlay values win over request variables, which win over file globals.
func Resolve(doc Document, req Request, overlay map[string]string) map[string]string {
	vals := make(map[string]string, len(doc.Globals)+len(req.Vars)+len(overlay))
	for _, v := range doc.Globals {
		vals[v.Name] = v.Value
	}
	for _, v := range req.Vars {
		vals[v.Name] = v.Value
	}
	maps.Copy(vals, overlay)
	expand(vals)
	return vals
}

// DisplayVars returns the variables that apply to a request, in stable order.
func DisplayVars(doc Document, req Request, overlay map[string]string) []Variable {
	values := Resolve(doc, req, overlay)
	names := unique(joinNames(namesOf(doc.Globals), namesOf(req.Vars), Referenced(req)))
	out := make([]Variable, 0, len(names))
	for _, name := range names {
		out = append(out, Variable{Name: name, Value: values[name]})
	}
	return out
}

// Referenced returns simple {{name}} identifiers used by the request.
func Referenced(req Request) []string {
	parts := make([]string, 0, 2+len(req.Headers)*2)
	parts = append(parts, req.URL)
	for _, h := range req.Headers {
		parts = append(parts, h.Name, h.Value)
	}
	parts = append(parts, req.Body)
	var names []string
	for _, part := range parts {
		names = append(names, referencedIn(part)...)
	}
	return unique(names)
}

// Prepare substitutes variables and applies host-prefixing for relative URLs.
func Prepare(req Request, values map[string]string) Prepared {
	prepared := Prepared{
		Name:   interpolate(req.DisplayName(), values),
		Method: interpolate(req.Method, values),
		URL:    interpolate(req.URL, values),
		Body:   unescape(interpolate(req.Body, values)),
	}
	if prepared.Method == "" {
		prepared.Method = "GET"
	}
	prepared.URL = applyHost(prepared.URL, values)
	for _, h := range req.Headers {
		prepared.Headers = append(prepared.Headers, Header{
			Name:  interpolate(h.Name, values),
			Value: unescape(interpolate(h.Value, values)),
		})
	}
	prepared.URL = unescape(prepared.URL)
	return prepared
}

func applyHost(url string, values map[string]string) string {
	host, ok := values["host"]
	if !ok || !strings.HasPrefix(url, "/") {
		return url
	}
	return strings.TrimRight(host, "/") + url
}

func expand(vals map[string]string) {
	for range expandPasses {
		changed := false
		for k, v := range vals {
			next := interpolate(v, vals)
			if next != v {
				vals[k] = next
				changed = true
			}
		}
		if !changed {
			return
		}
	}
}

func interpolate(s string, values map[string]string) string {
	return refPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := match[2 : len(match)-2]
		if value, ok := values[name]; ok {
			return value
		}
		return match
	})
}

func unescape(s string) string {
	s = strings.ReplaceAll(s, `\{\{`, "{{")
	s = strings.ReplaceAll(s, `\}\}`, "}}")
	return s
}

func referencedIn(s string) []string {
	matches := refPattern.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil
	}
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, m[1])
	}
	return names
}

func namesOf(vars []Variable) []string {
	names := make([]string, len(vars))
	for i, v := range vars {
		names[i] = v.Name
	}
	return names
}

func joinNames(groups ...[]string) []string {
	var n int
	for _, g := range groups {
		n += len(g)
	}
	out := make([]string, 0, n)
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

func unique(names []string) []string {
	seen := make(map[string]bool, len(names))
	out := make([]string, 0, len(names))
	for _, name := range names {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}
