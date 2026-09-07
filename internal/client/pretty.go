package client

import (
	"bytes"
	"encoding/json/jsontext"
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
)

var (
	keyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Cyan)
	stringStyle = lipgloss.NewStyle().Foreground(lipgloss.Green)
	numberStyle = lipgloss.NewStyle().Foreground(lipgloss.Yellow)
	boolStyle   = lipgloss.NewStyle().Foreground(lipgloss.Magenta)
	nullStyle   = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
)

// Body formats a response body, pretty-printing JSON when possible.
func Body(raw []byte, contentType string, color bool) string {
	if looksLikeJSON(raw, contentType) {
		formatted, err := jsontext.AppendFormat(nil, bytes.TrimSpace(raw), jsontext.WithIndent("  "), jsontext.Multiline(true))
		if err == nil {
			text := string(formatted)
			if color {
				return colorizeJSON(text)
			}
			return text
		}
	}
	return string(raw)
}

func looksLikeJSON(raw []byte, contentType string) bool {
	return contentTypeJSON(contentType) || jsonValue(raw)
}

func contentTypeJSON(contentType string) bool {
	ct := strings.ToLower(contentType)
	media, _, _ := strings.Cut(ct, ";")
	media = strings.TrimSpace(media)
	return strings.Contains(media, "json")
}

func jsonValue(raw []byte) bool {
	kind := jsontext.Value(bytes.TrimSpace(raw)).Kind()
	return kind == '{' || kind == '['
}

func colorizeJSON(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		switch {
		case s[i] == '"':
			str, next := readJSONString(s, i)
			if followedByColon(s, next) {
				b.WriteString(keyStyle.Render(str))
			} else {
				b.WriteString(stringStyle.Render(str))
			}
			i = next
		case isNumberStart(s, i):
			num, next := readNumber(s, i)
			b.WriteString(numberStyle.Render(num))
			i = next
		case hasLiteralPrefix(s, i, "true"), hasLiteralPrefix(s, i, "false"):
			lit, next := readLiteral(s, i)
			b.WriteString(boolStyle.Render(lit))
			i = next
		case hasLiteralPrefix(s, i, "null"):
			lit, next := readLiteral(s, i)
			b.WriteString(nullStyle.Render(lit))
			i = next
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

func readJSONString(s string, i int) (string, int) {
	j := i + 1
	for j < len(s) {
		if s[j] == '\\' {
			j += 2
			continue
		}
		if s[j] == '"' {
			return s[i : j+1], j + 1
		}
		j++
	}
	return s[i:], len(s)
}

func followedByColon(s string, i int) bool {
	for i < len(s) && isSpace(s[i]) {
		i++
	}
	return i < len(s) && s[i] == ':'
}

func isNumberStart(s string, i int) bool {
	if s[i] == '-' {
		return i+1 < len(s) && isDigit(s[i+1])
	}
	return isDigit(s[i])
}

func readNumber(s string, i int) (string, int) {
	j := i
	if s[j] == '-' {
		j++
	}
	for j < len(s) && isDigit(s[j]) {
		j++
	}
	if j < len(s) && s[j] == '.' {
		j++
		for j < len(s) && isDigit(s[j]) {
			j++
		}
	}
	if j < len(s) && (s[j] == 'e' || s[j] == 'E') {
		j++
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			j++
		}
		for j < len(s) && isDigit(s[j]) {
			j++
		}
	}
	return s[i:j], j
}

func hasLiteralPrefix(s string, i int, lit string) bool {
	if !strings.HasPrefix(s[i:], lit) {
		return false
	}
	end := i + len(lit)
	if end < len(s) && isIdentRune(rune(s[end])) {
		return false
	}
	return true
}

func readLiteral(s string, i int) (string, int) {
	switch {
	case strings.HasPrefix(s[i:], "true"):
		return s[i : i+4], i + 4
	case strings.HasPrefix(s[i:], "false"):
		return s[i : i+5], i + 5
	default:
		return s[i : i+4], i + 4
	}
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\n' || b == '\t' || b == '\r'
}

func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
