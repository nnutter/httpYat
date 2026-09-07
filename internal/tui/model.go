package tui

import (
	"maps"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"

	"github.com/nnutter/httpYat/internal/client"
	"github.com/nnutter/httpYat/internal/httpfile"
)

type focus int

const (
	focusRequests focus = iota
	focusVars
	focusResponse
)

const focusCount = 3

type status int

const (
	statusIdle status = iota
	statusLoading
	statusDone
	statusFailed
)

// entry identifies one request within the workspace docs.
type entry struct {
	doc int
	req int
}

// Model is the Bubble Tea application state.
type Model struct {
	docs    []httpfile.Document
	entries []entry
	sender  client.Sender
	timeout time.Duration

	selected  int
	varCursor int
	focus     focus
	reqOffset int
	varOffset int
	pendingG  bool
	overlay   map[string]string
	editing   bool
	showHeads bool
	seq       int

	status status
	result client.Result
	err    error
	notice string
	yank   func(string) error

	width  int
	height int

	input    textinput.Model
	viewport viewport.Model
	spinner  spinner.Model
	help     help.Model
	keys     keyMap
	styles   styles
}

// New constructs a TUI model for a parsed document.
func New(doc httpfile.Document, sender client.Sender, timeout time.Duration) Model {
	return NewWorkspace([]httpfile.Document{doc}, sender, timeout)
}

// NewWorkspace constructs a TUI model for multiple parsed documents.
func NewWorkspace(docs []httpfile.Document, sender client.Sender, timeout time.Duration) Model {
	in := textinput.New()
	in.Prompt = ""
	in.Placeholder = "value"
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	vp := viewport.New()
	vp.SoftWrap = true
	h := help.New()
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if sender == nil {
		sender = client.New(client.Options{Timeout: timeout})
	}
	entries := make([]entry, 0)
	for di := range docs {
		for ri := range docs[di].Requests {
			entries = append(entries, entry{doc: di, req: ri})
		}
	}
	return Model{
		docs:     docs,
		entries:  entries,
		sender:   sender,
		timeout:  timeout,
		yank:     clipboard.WriteAll,
		overlay:  map[string]string{},
		width:    80,
		height:   24,
		input:    in,
		viewport: vp,
		spinner:  sp,
		help:     h,
		keys:     newKeyMap(),
		styles:   newStyles(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) selectedEntry() (httpfile.Document, httpfile.Request, bool) {
	if m.selected < 0 || m.selected >= len(m.entries) {
		return httpfile.Document{}, httpfile.Request{}, false
	}
	e := m.entries[m.selected]
	return m.docs[e.doc], m.docs[e.doc].Requests[e.req], true
}

func (m Model) vars() []httpfile.Variable {
	doc, req, ok := m.selectedEntry()
	if !ok {
		if len(m.docs) == 0 {
			return httpfile.DisplayVars(httpfile.Document{}, httpfile.Request{}, m.overlay)
		}
		return httpfile.DisplayVars(m.docs[0], httpfile.Request{}, m.overlay)
	}
	return httpfile.DisplayVars(doc, req, m.overlay)
}

func (m Model) withOverlay(name, value string) Model {
	next := maps.Clone(m.overlay)
	if next == nil {
		next = map[string]string{}
	}
	next[name] = value
	m.overlay = next
	return m
}

func (m Model) clientTimeout() time.Duration {
	if m.timeout > 0 {
		return m.timeout
	}
	return 30 * time.Second
}
