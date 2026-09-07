package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/nnutter/httpYat/internal/client"
	"github.com/nnutter/httpYat/internal/httpfile"
)

func TestViewListsRequestsAndVariables(t *testing.T) {
	m := newTestModel(t, `@host = https://example.com
### List
GET {{host}}/users
### Create
POST {{host}}/users
`, nil)
	view := m.render()
	if !strings.Contains(view, "httpYat") {
		t.Fatalf("missing app title:\n%s", view)
	}
	if !strings.Contains(view, "List") || !strings.Contains(view, "Create") {
		t.Fatalf("missing requests:\n%s", view)
	}
	if !strings.Contains(view, "host") {
		t.Fatalf("missing variables:\n%s", view)
	}
}

func TestNavigationAndFocus(t *testing.T) {
	m := newTestModel(t, "GET https://example.com/a\n###\nGET https://example.com/b\n", nil)
	m, _ = apply(m, press("j"))
	if m.selected != 1 {
		t.Fatalf("selected = %d", m.selected)
	}
	m, _ = apply(m, press("k"))
	if m.selected != 0 {
		t.Fatalf("selected = %d", m.selected)
	}
	m, _ = apply(m, press("tab"))
	if m.focus != focusVars {
		t.Fatalf("focus = %d", m.focus)
	}
	m, _ = apply(m, press("tab"))
	if m.focus != focusResponse {
		t.Fatalf("focus = %d", m.focus)
	}
}

func TestEditVariableThenSend(t *testing.T) {
	var got client.Input
	src := "@host = https://example.com\n# @name getItem\n@id = 1\nGET {{host}}/items/{{id}}\n"
	m := newTestModel(t, src, client.SendFunc(func(_ context.Context, in client.Input) (client.Result, error) {
		got = in
		return client.Result{
			Status:      "200 OK",
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
			Body:        []byte(`{"ok":true,"id":3}`),
			Headers:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	}))
	m.docs[0].Path = "api.http"
	m, _ = apply(m, press("tab"))
	m, _ = apply(m, press("j"))
	m, cmd := apply(m, press("enter"))
	if !m.editing {
		t.Fatal("expected edit mode")
	}
	_ = cmd
	m.input.SetValue("3")
	m, _ = apply(m, press("enter"))
	if m.editing {
		t.Fatal("expected edit to commit")
	}
	if m.overlay["id"] != "3" {
		t.Fatalf("overlay = %v", m.overlay)
	}

	m.focus = focusRequests
	m, cmd = apply(m, press("enter"))
	m = applyMsgs(t, m, cmd)
	if m.status != statusDone {
		t.Fatalf("status = %d err=%v", m.status, m.err)
	}
	if got.File != "api.http" {
		t.Errorf("file = %q", got.File)
	}
	if got.Name != "getItem" {
		t.Errorf("name = %q", got.Name)
	}
	if got.Vars["id"] != "3" {
		t.Errorf("vars = %v", got.Vars)
	}
	view := m.render()
	if !strings.Contains(view, "200") {
		t.Fatalf("missing status:\n%s", view)
	}
	if !strings.Contains(view, `"ok"`) && !strings.Contains(view, "ok") {
		t.Fatalf("missing body:\n%s", view)
	}
}

func TestSendErrorAndHeadersToggle(t *testing.T) {
	m := newTestModel(t, "GET https://example.com\n", client.SendFunc(func(context.Context, client.Input) (client.Result, error) {
		return client.Result{
			Status:     "418 I'm a teapot",
			StatusCode: http.StatusTeapot,
			Headers: http.Header{
				"X-Trace":      []string{"abc"},
				"Content-Type": []string{"application/json"},
			},
			ContentType: "application/json",
			Body:        []byte(`{"err":"nope"}`),
		}, nil
	}))
	m, cmd := apply(m, press("r"))
	m = applyMsgs(t, m, cmd)
	if m.result.StatusCode != http.StatusTeapot {
		t.Fatalf("status = %d", m.result.StatusCode)
	}
	m, _ = apply(m, press("H"))
	if !m.showHeads {
		t.Fatal("expected headers visible")
	}
	body := renderResult(m)
	if !strings.Contains(body, "X-Trace") && !strings.Contains(strings.ToLower(body), "x-trace") {
		t.Fatalf("headers missing: %q", body)
	}
}

func TestQuitAndEmptyDocument(t *testing.T) {
	m := New(httpfile.Document{}, client.SendFunc(func(context.Context, client.Input) (client.Result, error) {
		return client.Result{}, errors.New("unused")
	}), time.Second)
	m = resize(m, 100, 30)
	view := m.render()
	if !strings.Contains(view, "no requests") {
		t.Fatalf("view = %s", view)
	}
	_, cmd := apply(m, press("q"))
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
	_, cmd = apply(m, press("enter"))
	if cmd != nil {
		t.Fatal("send with no requests should be a no-op")
	}
}

func TestCancelEdit(t *testing.T) {
	m := newTestModel(t, "@id = 1\nGET /{{id}}\n", nil)
	m, _ = apply(m, press("tab"))
	m, _ = apply(m, press("e"))
	m.input.SetValue("99")
	m, _ = apply(m, press("esc"))
	if m.editing {
		t.Fatal("still editing")
	}
	if m.overlay["id"] == "99" {
		t.Fatal("cancel should not save")
	}
}

func TestHelpToggleAndStaleResultIgnored(t *testing.T) {
	m := newTestModel(t, "GET https://example.com\n", nil)
	m, _ = apply(m, press("?"))
	if !m.help.ShowAll {
		t.Fatal("expected full help")
	}
	m.seq = 2
	m = applyResult(m, resultMsg{seq: 1, err: io.EOF})
	if m.status == statusFailed {
		t.Fatal("stale result should be ignored")
	}
}

func TestInitAndWindowSize(t *testing.T) {
	m := newTestModel(t, "GET https://example.com\n", nil)
	if cmd := m.Init(); cmd != nil {
		t.Fatal("init should be idle")
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	got := next.(Model)
	if got.width != 120 || got.height != 40 {
		t.Fatalf("size = %dx%d", got.width, got.height)
	}
}

func TestWorkspaceGroupsByFile(t *testing.T) {
	a := mustDoc(t, "GET https://example.com/a\n", "a.http")
	b := mustDoc(t, "GET https://example.com/b\n", "b.http")
	m := NewWorkspace([]httpfile.Document{a, b}, nil, time.Second)
	m = resize(m, 120, 36)
	view := m.render()
	ia := indexOf(view, "a.http")
	ib := indexOf(view, "b.http")
	if ia < 0 || ib < 0 || ib < ia {
		t.Fatalf("missing file headings in order:\n%s", view)
	}
	if len(m.entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(m.entries))
	}
}

func TestDuplicateNamesAcrossFilesSendCorrectFile(t *testing.T) {
	src := "# @name dup\nGET https://example.com/x\n"
	a := mustDoc(t, src, "a.http")
	b := mustDoc(t, src, "b.http")
	var got []client.Input
	m := NewWorkspace([]httpfile.Document{a, b}, client.SendFunc(func(_ context.Context, in client.Input) (client.Result, error) {
		got = append(got, in)
		return client.Result{StatusCode: http.StatusOK}, nil
	}), time.Second)
	m = resize(m, 120, 36)
	m, cmd := apply(m, press("enter"))
	m = applyMsgs(t, m, cmd)
	m, _ = apply(m, press("tab")) // results focus response; go back
	m, _ = apply(m, press("j"))
	m, cmd = apply(m, press("enter"))
	_ = applyMsgs(t, m, cmd)
	if len(got) != 2 {
		t.Fatalf("sends = %d, want 2", len(got))
	}
	if got[0].File != "a.http" || got[0].Name != "dup" {
		t.Errorf("first = %+v", got[0])
	}
	if got[1].File != "b.http" || got[1].Name != "dup" {
		t.Errorf("second = %+v", got[1])
	}
}

func TestYankCopiesPrettyBodyWithoutColor(t *testing.T) {
	m := newTestModel(t, "GET https://example.com\n", nil)
	raw := []byte(`{"b":[true,null],"a":1}`)
	m.status = statusDone
	m.result = client.Result{
		Status:      "200 OK",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        raw,
	}
	var copied string
	m.yank = func(s string) error {
		copied = s
		return nil
	}
	m, _ = apply(m, press("y"))
	want := client.Body(raw, "application/json", false)
	if copied != want {
		t.Fatalf("copied = %q, want %q", copied, want)
	}
	if strings.Contains(copied, "\x1b") {
		t.Fatalf("copied text should be plain, got %q", copied)
	}
	if !strings.Contains(m.notice, "copied") {
		t.Fatalf("notice = %q", m.notice)
	}
	if !strings.Contains(m.renderFooter(), "copy") {
		t.Fatalf("legend missing copy hint: %q", m.renderFooter())
	}
}

func TestYankWithoutResult(t *testing.T) {
	m := newTestModel(t, "GET https://example.com\n", nil)
	m.yank = func(string) error {
		t.Fatal("clipboard should not be called")
		return nil
	}
	m, _ = apply(m, press("y"))
	if m.notice != "nothing to copy" {
		t.Fatalf("notice = %q", m.notice)
	}
}

func TestYankFailureKeepsResult(t *testing.T) {
	m := newTestModel(t, "GET https://example.com\n", nil)
	m.status = statusDone
	m.result = client.Result{StatusCode: http.StatusOK, Body: []byte("hi")}
	m.yank = func(string) error { return errors.New("no pbcopy") }
	m, _ = apply(m, press("y"))
	if !strings.Contains(m.notice, "copy failed") {
		t.Fatalf("notice = %q", m.notice)
	}
	if m.status != statusDone {
		t.Fatalf("status = %d, want done", m.status)
	}
}

func TestFooterStaysVisibleAfterSend(t *testing.T) {
	m := newTestModel(t, "GET https://example.com/a\n###\nGET https://example.com/b\n", client.SendFunc(func(context.Context, client.Input) (client.Result, error) {
		return client.Result{Status: "200 OK", StatusCode: http.StatusOK, ContentType: "application/json", Body: []byte(`{"ok":true}`)}, nil
	}))
	m = resize(m, 120, 24)
	assertFits(t, m.render(), 24)
	m, cmd := apply(m, press("enter"))
	m = applyMsgs(t, m, cmd)
	view := m.render()
	assertFits(t, view, 24)
	if !strings.Contains(view, "copy") {
		t.Fatalf("legend missing copy hint:\n%s", view)
	}
}

func TestLongRequestListScrollsWithinPane(t *testing.T) {
	var b strings.Builder
	for i := range 30 {
		fmt.Fprintf(&b, "GET https://example.com/%02d\n###\n", i)
	}
	m := newTestModel(t, b.String(), nil)
	m = resize(m, 120, 24)
	assertFits(t, m.render(), 24)
	for range 40 {
		m, _ = apply(m, press("j"))
	}
	if m.selected != 29 {
		t.Fatalf("selected = %d, want 29", m.selected)
	}
	view := m.render()
	assertFits(t, view, 24)
	if !strings.Contains(view, "/29") {
		t.Fatalf("selected row not visible:\n%s", view)
	}
	if strings.Contains(view, "/00") {
		t.Fatalf("window should have scrolled past first row:\n%s", view)
	}
}

func TestManyVarsStayWithinPane(t *testing.T) {
	var b strings.Builder
	for i := range 25 {
		fmt.Fprintf(&b, "@v%02d = %d\n", i, i)
	}
	b.WriteString("GET https://example.com/{{v00}}\n")
	m := newTestModel(t, b.String(), nil)
	m = resize(m, 120, 24)
	assertFits(t, m.render(), 24)
	m, _ = apply(m, press("tab"))
	for range 30 {
		m, _ = apply(m, press("j"))
	}
	view := m.render()
	assertFits(t, view, 24)
	if !strings.Contains(view, "v24") {
		t.Fatalf("cursor row not visible:\n%s", view)
	}
}

func assertFits(t *testing.T, view string, height int) {
	t.Helper()
	if h := lipgloss.Height(view); h > height {
		t.Fatalf("view height %d exceeds %d:\n%s", h, height, view)
	}
}

func TestResultFocusesResponse(t *testing.T) {
	m := newTestModel(t, "GET https://example.com\n", client.SendFunc(func(context.Context, client.Input) (client.Result, error) {
		return client.Result{StatusCode: http.StatusOK}, nil
	}))
	m.focus = focusRequests
	m, cmd := apply(m, press("enter"))
	m = applyMsgs(t, m, cmd)
	if m.status != statusDone {
		t.Fatalf("status = %d", m.status)
	}
	if m.focus != focusResponse {
		t.Fatalf("focus = %d, want response", m.focus)
	}
}

func TestFailedResultFocusesResponse(t *testing.T) {
	m := newTestModel(t, "GET https://example.com\n", client.SendFunc(func(context.Context, client.Input) (client.Result, error) {
		return client.Result{}, errors.New("boom")
	}))
	m.focus = focusRequests
	m, cmd := apply(m, press("enter"))
	m = applyMsgs(t, m, cmd)
	if m.status != statusFailed {
		t.Fatalf("status = %d", m.status)
	}
	if m.focus != focusResponse {
		t.Fatalf("focus = %d, want response", m.focus)
	}
}

func TestStaleResultKeepsFocus(t *testing.T) {
	m := newTestModel(t, "GET https://example.com\n", nil)
	m.focus = focusRequests
	m.seq = 2
	m = applyResult(m, resultMsg{seq: 1, result: client.Result{StatusCode: http.StatusOK}})
	if m.focus != focusRequests {
		t.Fatalf("focus = %d, want requests", m.focus)
	}
}

func TestFooterLegendContextualEditHint(t *testing.T) {
	m := newTestModel(t, "GET https://example.com\n", nil)
	m.focus = focusRequests
	if got := m.renderFooter(); !strings.Contains(got, "send") {
		t.Fatalf("requests legend missing send hint: %q", got)
	}
	m.focus = focusVars
	got := m.renderFooter()
	if !strings.Contains(got, "enter") || !strings.Contains(got, "edit") {
		t.Fatalf("vars legend missing enter edit hint: %q", got)
	}
	if strings.Contains(got, "send") {
		t.Fatalf("vars legend must not promise send on enter: %q", got)
	}
}

func TestPaneMovementWithHL(t *testing.T) {
	m := newTestModel(t, "GET https://example.com\n", nil)
	if m.focus != focusRequests {
		t.Fatalf("focus = %d", m.focus)
	}
	m, _ = apply(m, press("l"))
	if m.focus != focusVars {
		t.Fatalf("focus = %d, want vars", m.focus)
	}
	m, _ = apply(m, press("l"))
	if m.focus != focusResponse {
		t.Fatalf("focus = %d, want response", m.focus)
	}
	m, _ = apply(m, press("l"))
	if m.focus != focusRequests {
		t.Fatalf("focus = %d, want wrap to requests", m.focus)
	}
	m, _ = apply(m, press("h"))
	if m.focus != focusResponse {
		t.Fatalf("focus = %d, want wrap to response", m.focus)
	}
}

func longListModel(t *testing.T, n int) Model {
	t.Helper()
	var b strings.Builder
	for i := range n {
		fmt.Fprintf(&b, "GET https://example.com/%02d\n###\n", i)
	}
	m := newTestModel(t, b.String(), nil)
	return resize(m, 120, 24)
}

func TestFirstLastRequests(t *testing.T) {
	m := longListModel(t, 30)
	m, _ = apply(m, press("G"))
	if m.selected != 29 {
		t.Fatalf("selected = %d, want 29", m.selected)
	}
	view := m.render()
	assertFits(t, view, 24)
	if !strings.Contains(view, "/29") {
		t.Fatalf("last row not visible:\n%s", view)
	}
	m, _ = apply(m, press("g"))
	if m.selected != 29 {
		t.Fatalf("single g must not move: selected = %d", m.selected)
	}
	m, _ = apply(m, press("g"))
	if m.selected != 0 {
		t.Fatalf("selected = %d, want 0", m.selected)
	}
	view = m.render()
	assertFits(t, view, 24)
	if !strings.Contains(view, "/00") {
		t.Fatalf("first row not visible:\n%s", view)
	}
}

func TestFirstLastResponse(t *testing.T) {
	m := newTestModel(t, "GET https://example.com\n", nil)
	m.status = statusDone
	m.result = client.Result{Body: []byte(strings.Repeat("line\n", 100))}
	m = refreshResult(m)
	m = resize(m, 120, 24)
	m.focus = focusResponse
	m, _ = apply(m, press("G"))
	if !m.viewport.AtBottom() {
		t.Fatal("expected viewport at bottom")
	}
	m, _ = apply(m, press("g"))
	m, _ = apply(m, press("g"))
	if !m.viewport.AtTop() {
		t.Fatal("expected viewport at top")
	}
}

func TestPagingRequests(t *testing.T) {
	m := longListModel(t, 30)
	rows := listRows(requestHeight(m.bodyHeight(), m.varsHeight(m.bodyHeight())))
	m, _ = apply(m, press("ctrl+f"))
	if m.selected != rows {
		t.Fatalf("selected = %d, want %d", m.selected, rows)
	}
	assertFits(t, m.render(), 24)
	m, _ = apply(m, press("ctrl+b"))
	if m.selected != 0 {
		t.Fatalf("selected = %d, want 0", m.selected)
	}
}

func TestPagingResponse(t *testing.T) {
	m := newTestModel(t, "GET https://example.com\n", nil)
	m.status = statusDone
	m.result = client.Result{Body: []byte(strings.Repeat("line\n", 100))}
	m = refreshResult(m)
	m = resize(m, 120, 24)
	m.focus = focusResponse
	for range 10 {
		m, _ = apply(m, press("ctrl+f"))
		if m.viewport.AtBottom() {
			break
		}
	}
	if !m.viewport.AtBottom() {
		t.Fatal("expected viewport at bottom after paging down")
	}
	for range 10 {
		m, _ = apply(m, press("ctrl+b"))
		if m.viewport.AtTop() {
			break
		}
	}
	if !m.viewport.AtTop() {
		t.Fatal("expected viewport at top after paging up")
	}
	assertFits(t, m.render(), 24)
}

func click(x, y int) tea.MouseClickMsg {
	return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft}
}

func TestClickSelectsPane(t *testing.T) {
	m := newTestModel(t, "@id = 1\nGET https://example.com/{{id}}\n", nil)
	m.focus = focusRequests
	m, _ = apply(m, click(60, 10))
	if m.focus != focusResponse {
		t.Fatalf("focus = %d, want response", m.focus)
	}
	m, _ = apply(m, click(5, 33))
	if m.focus != focusVars {
		t.Fatalf("focus = %d, want vars", m.focus)
	}
	m, _ = apply(m, click(5, 5))
	if m.focus != focusRequests {
		t.Fatalf("focus = %d, want requests", m.focus)
	}
}

func TestClickIgnoresChromeAndRightButton(t *testing.T) {
	m := newTestModel(t, "GET https://example.com\n", nil)
	m.focus = focusRequests
	m, _ = apply(m, click(5, 0))
	m, _ = apply(m, click(5, 35))
	m, _ = apply(m, tea.MouseClickMsg{X: 60, Y: 10, Button: tea.MouseRight})
	if m.focus != focusRequests {
		t.Fatalf("focus = %d, want unchanged requests", m.focus)
	}
}

func TestClickIgnoredWhileEditing(t *testing.T) {
	m := newTestModel(t, "@id = 1\nGET https://example.com/{{id}}\n", nil)
	m, _ = apply(m, press("tab"))
	m, _ = apply(m, press("e"))
	if !m.editing {
		t.Fatal("expected edit mode")
	}
	m, _ = apply(m, click(60, 10))
	if m.focus != focusVars || !m.editing {
		t.Fatalf("focus = %d editing = %v, want vars edit intact", m.focus, m.editing)
	}
}

func TestGotoBottomReachesFinalLine(t *testing.T) {
	var b strings.Builder
	for i := range 60 {
		fmt.Fprintf(&b, "line %02d\n", i)
	}
	m := newTestModel(t, "GET https://example.com\n", nil)
	m = resize(m, 171, 50)
	m.seq = 1
	m, _ = apply(m, resultMsg{seq: 1, result: client.Result{StatusCode: http.StatusOK, Body: []byte(b.String())}})
	m, _ = apply(m, press("G"))
	view := m.render()
	assertFits(t, view, 50)
	if !strings.Contains(view, "line 59") {
		t.Fatalf("bottom line not reachable:\n%s", view)
	}
}

func mustDoc(t *testing.T, src, path string) httpfile.Document {
	t.Helper()
	doc, err := httpfile.Parse(src, "")
	if err != nil {
		t.Fatal(err)
	}
	doc.Path = path
	return doc
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func newTestModel(t *testing.T, src string, sender client.Sender) Model {
	t.Helper()
	doc, err := httpfile.Parse(src, "")
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		sender = client.SendFunc(func(context.Context, client.Input) (client.Result, error) {
			return client.Result{}, errors.New("unexpected send")
		})
	}
	m := New(doc, sender, time.Second)
	return resize(m, 120, 36)
}

func press(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	case "ctrl+f":
		return tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl}
	case "ctrl+b":
		return tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl}
	default:
		r := []rune(s)
		return tea.KeyPressMsg{Text: s, Code: r[0]}
	}
}

func applyMsgs(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for _, msg := range flatten(cmd) {
		if time.Now().After(deadline) {
			t.Fatal("timeout applying commands")
		}
		var next tea.Cmd
		m, next = apply(m, msg)
		_ = next
	}
	return m
}

func flatten(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	var out []tea.Msg
	for _, c := range batch {
		out = append(out, flatten(c)...)
	}
	return out
}
