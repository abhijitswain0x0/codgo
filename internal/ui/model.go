// Package ui is the Bubble Tea front end.
package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/abhijitswain0x0/codgo/internal/oc"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	c   *oc.Client
	th  Theme
	cwd string

	w, h int
	vp   viewport.Model
	ta   textarea.Model
	sp   spinner.Model

	blocks []*Block
	byPart map[string]*Block
	roles  map[string]string // message id -> role

	sid       string
	model     oc.ModelRef
	busy      bool
	busySince time.Time
	streamUp  bool
	ready     bool
	expanded  bool
	perm      []oc.PermissionAsked // pending approvals, oldest first
	permSel   int
	pick      *picker
	popSel    int
	help      bool
	sawDown   bool
	provs     oc.Providers
	hist      []string
	histIdx   int // len(hist) means a fresh line
}

func New(c *oc.Client, cwd string, th Theme, model string) *Model {
	ta := textarea.New()
	ta.Prompt = "> "
	ta.Placeholder = ""
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.SetHeight(1)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(th.Accent)
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(th.Dim)
	ta.KeyMap.InsertNewline.SetKeys("alt+enter", "ctrl+j", "shift+enter")
	ta.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Spinner{Frames: []string{"|", "/", "-", "\\"}, FPS: time.Second / 6}
	sp.Style = lipgloss.NewStyle().Foreground(th.Accent)

	m := &Model{c: c, th: th, cwd: cwd, ta: ta, sp: sp,
		byPart: map[string]*Block{}, roles: map[string]string{}}
	m.hist = loadHistory()
	m.histIdx = len(m.hist)
	if i := strings.Index(model, "/"); i > 0 {
		m.model = oc.ModelRef{ProviderID: model[:i], ModelID: model[i+1:]}
	}
	return m
}

func (m *Model) add(b *Block) *Block {
	b.dirty = true
	m.blocks = append(m.blocks, b)
	return b
}

func (m *Model) modelLabel() string {
	if m.model.ModelID == "" {
		return "default model"
	}
	return m.model.ProviderID + "/" + m.model.ModelID
}

func (m *Model) shortSID() string {
	if m.sid == "" {
		return "no session"
	}
	if len(m.sid) > 12 {
		return m.sid[:12]
	}
	return m.sid
}

func (m *Model) elapsed() string {
	return fmt.Sprintf("%ds", int(time.Since(m.busySince).Seconds()))
}

func bg() context.Context { return context.Background() }

func newVP(w, h int) viewport.Model {
	v := viewport.New(w, h)
	v.MouseWheelEnabled = true
	return v
}
