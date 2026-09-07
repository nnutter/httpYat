package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/nnutter/httpYat/internal/client"
)

// View implements tea.Model.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m Model) render() string {
	header := m.renderHeader()
	footer := m.renderFooter()
	bodyH := max(m.height-lipgloss.Height(header)-lipgloss.Height(footer), 1)
	leftW, rightW, _ := panes(m)
	left := lipgloss.JoinVertical(lipgloss.Left,
		m.renderRequests(leftW, requestHeight(bodyH, m.varsHeight(bodyH))),
		m.renderVars(leftW, m.varsHeight(bodyH)),
	)
	right := m.renderResponse(rightW, bodyH)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func panes(m Model) (left, right, body int) {
	body = max(m.height-2, 8)
	left = min(42, max(m.width/3, 24))
	right = max(m.width-left, 20)
	return left, right, body
}

func requestHeight(bodyH, varsH int) int {
	return max(bodyH-varsH, 5)
}

func (m Model) varsHeight(bodyH int) int {
	n := max(len(m.vars()), 1) + 3 // title + rows + pane border
	limit := max(bodyH/3, 4)
	return min(n, limit)
}

// focusHelp keeps the shortcut legend truthful per pane: with the
// variables pane focused Enter edits instead of sending, so the send
// hint (also Enter) is hidden there in favor of the edit hint.
type focusHelp struct {
	keys  keyMap
	focus focus
}

func (f focusHelp) ShortHelp() []key.Binding {
	keys := f.keys
	if f.focus == focusVars {
		keys.Edit.SetHelp("enter", "edit var")
		keys.Send.SetEnabled(false)
	}
	return keys.ShortHelp()
}

func (f focusHelp) FullHelp() [][]key.Binding {
	return f.keys.FullHelp()
}

func (m Model) renderFooter() string {
	legend := m.help.View(focusHelp{keys: m.keys, focus: m.focus})
	if m.notice == "" {
		return legend
	}
	style := m.styles.muted
	if !strings.HasPrefix(m.notice, "copied") {
		style = m.styles.statusFail
	}
	return legend + "\n" + style.Render(m.notice)
}

func (m Model) bodyHeight() int {
	return max(m.height-lipgloss.Height(m.renderHeader())-lipgloss.Height(m.renderFooter()), 1)
}

// responseViewportHeight shares a bodyH-tall pane between the border,
// title, status line, and viewport so the footer stays on screen.
func responseViewportHeight(bodyH int, st status) int {
	const chrome = 3 // pane border (2) + title (1)
	extra := 0
	if st == statusDone || st == statusFailed || st == statusLoading {
		extra = 1 // status, spinner, or error line
	}
	return max(bodyH-chrome-extra, 1)
}

// listRows returns how many body rows fit in a height-tall pane under
// its title.
func listRows(height int) int {
	return max(height-3, 1)
}

// windowRange returns the [start, end) slice of n rows fitting cap
// lines starting from offset.
func windowRange(n, offset, cap int) (int, int) {
	if n == 0 || cap <= 0 {
		return 0, 0
	}
	start := min(max(offset, 0), n-1)
	return start, min(start+cap, n)
}

// windowEntries returns the [start, end) entries fitting cap lines,
// counting one line per request plus a heading per file group.
func windowEntries(entries []entry, grouped bool, offset, cap int) (int, int) {
	n := len(entries)
	if n == 0 || cap <= 0 {
		return 0, 0
	}
	start := min(max(offset, 0), n-1)
	used := 0
	lastDoc := -1
	if grouped {
		used = 1 // sticky heading for the first visible group
		lastDoc = entries[start].doc
	}
	end := start
	for end < n {
		need := 1
		if grouped && entries[end].doc != lastDoc {
			need = 2 // new group heading + request
		}
		if used+need > cap {
			break
		}
		used += need
		if grouped {
			lastDoc = entries[end].doc
		}
		end++
	}
	if end == start {
		end = start + 1 // cap too small: show the row anyway
	}
	return start, end
}

func (m Model) renderHeader() string {
	path := m.headerPath()
	count := fmt.Sprintf("%d requests", len(m.entries))
	title := m.styles.header.Render("httpYat")
	meta := m.styles.muted.Render("  " + path + "  " + count)
	line := title + meta
	return ansi.Truncate(line, m.width, "…")
}

func (m Model) headerPath() string {
	if len(m.docs) == 1 {
		if m.docs[0].Path == "" {
			return "untitled.http"
		}
		return m.docs[0].Path
	}
	if len(m.docs) == 0 {
		return "untitled.http"
	}
	dir := filepath.Dir(m.docs[0].Path)
	for _, d := range m.docs[1:] {
		if filepath.Dir(d.Path) != dir {
			return fmt.Sprintf("%d files", len(m.docs))
		}
	}
	if dir == "" || dir == "." {
		return fmt.Sprintf("%d files", len(m.docs))
	}
	return dir
}

func (m Model) renderRequests(width, height int) string {
	style := m.paneStyle(focusRequests).Width(width - 2).Height(height - 2)
	var b strings.Builder
	b.WriteString(m.styles.title.Render("Requests"))
	b.WriteByte('\n')
	if len(m.entries) == 0 {
		b.WriteString(m.styles.muted.Render("no requests"))
		return style.Render(b.String())
	}
	grouped := len(m.docs) > 1
	rows := listRows(height)
	start, end := windowEntries(m.entries, grouped, m.reqOffset, rows)
	lastDoc := -1
	for i := start; i < end; i++ {
		e := m.entries[i]
		if grouped && e.doc != lastDoc {
			lastDoc = e.doc
			b.WriteString(m.styles.muted.Render(displayFile(m.docs[e.doc].Path)))
			b.WriteByte('\n')
		}
		req := m.docs[e.doc].Requests[e.req]
		cursor := "  "
		if i == m.selected {
			cursor = m.styles.cursor.Render("> ")
		}
		method := m.styles.methodStyle(req.Method).Render(req.Method)
		name := req.DisplayName()
		if after, ok := strings.CutPrefix(name, req.Method+" "); ok {
			name = after
		}
		line := cursor + method + " " + name
		if i == m.selected && m.focus == focusRequests {
			line = m.styles.cursor.Render(ansi.Truncate(stripForWidth(line), width-6, "…"))
		} else {
			line = ansi.Truncate(line, width-6, "…")
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return style.Render(strings.TrimRight(b.String(), "\n"))
}

func (m Model) renderVars(width, height int) string {
	style := m.paneStyle(focusVars).Width(width - 2).Height(height - 2)
	var b strings.Builder
	b.WriteString(m.styles.title.Render("Variables"))
	b.WriteByte('\n')
	vars := m.vars()
	if len(vars) == 0 {
		b.WriteString(m.styles.muted.Render("none"))
		return style.Render(b.String())
	}
	rows := listRows(height)
	start, end := windowRange(len(vars), m.varOffset, rows)
	for k := start; k < end; k++ {
		v := vars[k]
		cursor := "  "
		if k == m.varCursor {
			cursor = m.styles.cursor.Render("> ")
		}
		value := v.Value
		if m.editing && k == m.varCursor {
			value = m.input.View()
		} else if value == "" {
			value = m.styles.muted.Render("unset")
		}
		line := cursor + v.Name + " = " + value
		b.WriteString(ansi.Truncate(line, width-6, "…"))
		b.WriteByte('\n')
	}
	return style.Render(strings.TrimRight(b.String(), "\n"))
}

func (m Model) renderResponse(width, height int) string {
	style := m.paneStyle(focusResponse).Width(width - 2).Height(height - 2)
	var b strings.Builder
	b.WriteString(m.styles.title.Render("Response"))
	b.WriteByte('\n')
	switch m.status {
	case statusIdle:
		b.WriteString(m.styles.muted.Render("press enter to send"))
	case statusLoading:
		b.WriteString(m.spinner.View() + " sending")
	case statusFailed:
		b.WriteString(m.styles.statusFail.Render("error"))
		b.WriteByte('\n')
		b.WriteString(m.viewport.View())
	case statusDone:
		b.WriteString(m.statusLine())
		b.WriteByte('\n')
		b.WriteString(m.viewport.View())
	}
	return style.Render(b.String())
}

func (m Model) statusLine() string {
	st := m.styles.statusOK
	if m.result.StatusCode >= 400 || m.result.StatusCode == 0 {
		st = m.styles.statusFail
	}
	parts := []string{
		st.Render(m.result.Status),
		m.styles.muted.Render(m.result.Duration.Truncate(time.Millisecond).String()),
	}
	if m.result.ContentType != "" {
		parts = append(parts, m.styles.muted.Render(m.result.ContentType))
	}
	if m.result.Truncated {
		parts = append(parts, m.styles.statusFail.Render("truncated"))
	}
	return strings.Join(parts, "  ")
}

func renderResult(m Model) string {
	var b strings.Builder
	if m.showHeads {
		for name, values := range m.result.Headers {
			fmt.Fprintf(&b, "%s: %s\n", name, strings.Join(values, ", "))
		}
		b.WriteByte('\n')
	}
	b.WriteString(client.Body(m.result.Body, m.result.ContentType, true))
	return b.String()
}

func displayFile(path string) string {
	if path == "" {
		return "untitled.http"
	}
	return filepath.Base(path)
}

func (m Model) paneStyle(f focus) lipgloss.Style {
	if m.focus == f {
		return m.styles.active
	}
	return m.styles.border
}

func stripForWidth(s string) string {
	return s
}
