package tui

import (
	"context"
	"fmt"
	"maps"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/nnutter/httpYat/internal/client"
)

type resultMsg struct {
	seq    int
	result client.Result
	err    error
}

// Update implements tea.Model as a thin wrapper around apply.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return apply(m, msg)
}

func apply(m Model, msg tea.Msg) (Model, tea.Cmd) {
	m, cmd := dispatch(m, msg)
	return fitViewport(m), cmd
}

// fitViewport keeps the stored viewport size identical to what render
// displays, so scroll math (GotoBottom, paging, AtBottom) agrees with
// the visible window. Status, header, and footer heights all vary, so
// this runs after every state change rather than only on resize.
func fitViewport(m Model) Model {
	_, right, _ := panes(m)
	m.viewport.SetWidth(max(right-4, 8))
	m.viewport.SetHeight(responseViewportHeight(m.bodyHeight(), m.status))
	return m
}

func dispatch(m Model, msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return resize(m, msg.Width, msg.Height), nil
	case resultMsg:
		return applyResult(m, msg), nil
	case spinner.TickMsg:
		if m.status != statusLoading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.MouseClickMsg:
		return clickPane(m, msg), nil
	case tea.KeyPressMsg:
		return handleKey(m, msg)
	default:
		if m.editing {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		if m.focus == focusResponse {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		return m, nil
	}
}

// clickPane focuses the pane under a mouse click, using the same
// geometry as render. Clicks on the header, footer, or while editing
// are ignored, as are non-left buttons.
func clickPane(m Model, msg tea.MouseClickMsg) Model {
	if m.editing || msg.Button != tea.MouseLeft {
		return m
	}
	headerH := lipgloss.Height(m.renderHeader())
	bodyH := m.bodyHeight()
	if msg.Y < headerH || msg.Y >= headerH+bodyH {
		return m
	}
	leftW, _, _ := panes(m)
	if msg.X >= leftW {
		m.focus = focusResponse
		return m
	}
	if msg.Y < headerH+requestHeight(bodyH, m.varsHeight(bodyH)) {
		m.focus = focusRequests
	} else {
		m.focus = focusVars
	}
	return m
}

func resize(m Model, width, height int) Model {
	m.width = max(width, 40)
	m.height = max(height, 12)
	m.help.SetWidth(m.width)
	m = layout(m)
	return m
}

func layout(m Model) Model {
	left, _, _ := panes(m)
	m = fitViewport(m)
	m.input.SetWidth(max(left-12, 6))
	return m
}

func handleKey(m Model, msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.editing {
		return handleEditKey(m, msg)
	}
	if !key.Matches(msg, m.keys.Yank) {
		m.notice = ""
	}
	if !key.Matches(msg, m.keys.First) && !key.Matches(msg, m.keys.Last) {
		m.pendingG = false
	}
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case key.Matches(msg, m.keys.Focus):
		m.focus = (m.focus + 1) % focusCount
		return m, nil
	case key.Matches(msg, m.keys.Headers):
		m.showHeads = !m.showHeads
		return refreshResult(m), nil
	case key.Matches(msg, m.keys.Yank):
		return yankBody(m), nil
	case key.Matches(msg, m.keys.Edit):
		return startEdit(m)
	case key.Matches(msg, m.keys.Send):
		if sendOnEnter(m, msg) {
			return send(m)
		}
		return startEdit(m)
	case key.Matches(msg, m.keys.Left):
		m.focus = (m.focus + focusCount - 1) % focusCount
		return m, nil
	case key.Matches(msg, m.keys.Right):
		m.focus = (m.focus + 1) % focusCount
		return m, nil
	case key.Matches(msg, m.keys.First):
		if m.pendingG {
			m.pendingG = false
			return goTop(m), nil
		}
		m.pendingG = true
		return m, nil
	case key.Matches(msg, m.keys.Last):
		m.pendingG = false
		return goBottom(m), nil
	case key.Matches(msg, m.keys.PageDown):
		return page(m, 1), nil
	case key.Matches(msg, m.keys.PageUp):
		return page(m, -1), nil
	case key.Matches(msg, m.keys.Up):
		return move(m, -1), nil
	case key.Matches(msg, m.keys.Down):
		return move(m, 1), nil
	default:
		if m.focus == focusResponse {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		return m, nil
	}
}

func sendOnEnter(m Model, msg tea.KeyPressMsg) bool {
	if msg.String() == "r" {
		return true
	}
	return m.focus != focusVars
}

func handleEditKey(m Model, msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.editing = false
		m.input.Blur()
		return m, nil
	case key.Matches(msg, m.keys.Confirm):
		return commitEdit(m), nil
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func move(m Model, delta int) Model {
	switch m.focus {
	case focusRequests:
		n := len(m.entries)
		if n == 0 {
			return m
		}
		m.selected = clampIndex(m.selected+delta, n)
		m.varCursor = 0
		m.reqOffset = fitEntryOffset(m, m.reqOffset, m.selected)
	case focusVars:
		n := len(m.vars())
		if n == 0 {
			return m
		}
		m.varCursor = clampIndex(m.varCursor+delta, n)
		m.varOffset = fitVarOffset(m, m.varOffset, m.varCursor)
	case focusResponse:
		if delta > 0 {
			m.viewport.ScrollDown(1)
		} else {
			m.viewport.ScrollUp(1)
		}
	}
	return m
}

func clampIndex(i, n int) int {
	return min(n-1, max(0, i))
}

func goTop(m Model) Model {
	switch m.focus {
	case focusRequests:
		if len(m.entries) == 0 {
			return m
		}
		m.selected = 0
		m.varCursor = 0
		m.reqOffset = fitEntryOffset(m, m.reqOffset, m.selected)
	case focusVars:
		if len(m.vars()) == 0 {
			return m
		}
		m.varCursor = 0
		m.varOffset = fitVarOffset(m, m.varOffset, m.varCursor)
	case focusResponse:
		m.viewport.GotoTop()
	}
	return m
}

func goBottom(m Model) Model {
	switch m.focus {
	case focusRequests:
		if len(m.entries) == 0 {
			return m
		}
		m.selected = len(m.entries) - 1
		m.varCursor = 0
		m.reqOffset = fitEntryOffset(m, m.reqOffset, m.selected)
	case focusVars:
		vars := m.vars()
		if len(vars) == 0 {
			return m
		}
		m.varCursor = len(vars) - 1
		m.varOffset = fitVarOffset(m, m.varOffset, m.varCursor)
	case focusResponse:
		m.viewport.GotoBottom()
	}
	return m
}

func page(m Model, dir int) Model {
	switch m.focus {
	case focusRequests:
		if len(m.entries) == 0 {
			return m
		}
		rows := listRows(requestHeight(m.bodyHeight(), m.varsHeight(m.bodyHeight())))
		m.selected = clampIndex(m.selected+dir*rows, len(m.entries))
		m.varCursor = 0
		m.reqOffset = fitEntryOffset(m, m.reqOffset, m.selected)
	case focusVars:
		vars := m.vars()
		if len(vars) == 0 {
			return m
		}
		rows := listRows(m.varsHeight(m.bodyHeight()))
		m.varCursor = clampIndex(m.varCursor+dir*rows, len(vars))
		m.varOffset = fitVarOffset(m, m.varOffset, m.varCursor)
	case focusResponse:
		if dir > 0 {
			m.viewport.PageDown()
		} else {
			m.viewport.PageUp()
		}
	}
	return m
}

func fitEntryOffset(m Model, offset, selected int) int {
	if len(m.entries) == 0 {
		return 0
	}
	rows := listRows(requestHeight(m.bodyHeight(), m.varsHeight(m.bodyHeight())))
	grouped := len(m.docs) > 1
	if selected < offset {
		return selected
	}
	for {
		_, end := windowEntries(m.entries, grouped, offset, rows)
		if selected < end {
			return offset
		}
		offset++
	}
}

func fitVarOffset(m Model, offset, cursor int) int {
	rows := listRows(m.varsHeight(m.bodyHeight()))
	if cursor < offset {
		return cursor
	}
	for cursor >= offset+rows {
		offset++
	}
	return offset
}

func startEdit(m Model) (Model, tea.Cmd) {
	vars := m.vars()
	if len(vars) == 0 {
		return m, nil
	}
	if m.focus != focusVars {
		m.focus = focusVars
	}
	current := vars[m.varCursor]
	m.editing = true
	m.input.SetValue(current.Value)
	m.input.CursorEnd()
	return m, m.input.Focus()
}

func commitEdit(m Model) Model {
	vars := m.vars()
	m.editing = false
	m.input.Blur()
	if len(vars) == 0 {
		return m
	}
	return m.withOverlay(vars[m.varCursor].Name, m.input.Value())
}

func send(m Model) (Model, tea.Cmd) {
	doc, req, ok := m.selectedEntry()
	if !ok {
		return m, nil
	}
	m.seq++
	m.status = statusLoading
	m.err = nil
	m.result = client.Result{}
	seq := m.seq
	sender := m.sender
	timeout := m.clientTimeout()
	in := client.Input{
		File:    doc.Path,
		Line:    req.Line,
		Name:    req.MetaName(),
		Vars:    maps.Clone(m.overlay),
		Timeout: timeout,
	}
	return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		res, err := sender.Send(ctx, in)
		return resultMsg{seq: seq, result: res, err: err}
	})
}

func yankBody(m Model) Model {
	if m.status != statusDone {
		m.notice = "nothing to copy"
		return m
	}
	if m.yank == nil {
		m.notice = "clipboard unavailable"
		return m
	}
	text := client.Body(m.result.Body, m.result.ContentType, false)
	if err := m.yank(text); err != nil {
		m.notice = "copy failed: " + err.Error()
		return m
	}
	m.notice = fmt.Sprintf("copied %d bytes", len(text))
	return m
}

func applyResult(m Model, msg resultMsg) Model {
	if msg.seq != m.seq {
		return m
	}
	m.focus = focusResponse
	if msg.err != nil {
		m.status = statusFailed
		m.err = msg.err
		m.viewport.SetContent(msg.err.Error())
		return m
	}
	m.status = statusDone
	m.err = nil
	m.result = msg.result
	return refreshResult(m)
}

func refreshResult(m Model) Model {
	if m.status != statusDone {
		return m
	}
	m.viewport.SetContent(renderResult(m))
	m.viewport.GotoTop()
	return m
}
