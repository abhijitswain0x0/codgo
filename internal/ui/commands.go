package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/abhijitswain0x0/codgo/internal/oc"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type cmdDef struct{ name, desc string }

var slashCmds = []cmdDef{
	{"/new", "start a new session"},
	{"/sessions", "pick and resume a session"},
	{"/model", "list and switch the model"},
	{"/clear", "clear the transcript view"},
	{"/quit", "exit codgo"},
}

type (
	sessionsMsg struct{ s []oc.Session }
	resumeMsg   struct {
		s    oc.Session
		msgs []oc.Message
	}
)

// picker is a small list that replaces the input box (sessions, models).
type picker struct {
	title string
	kind  string // "session" or "model"
	items []pickItem
	sel   int
}

type pickItem struct{ label, value string }

// popupItems returns slash commands matching what is typed so far.
func (m *Model) popupItems() []cmdDef {
	v := m.ta.Value()
	if !strings.HasPrefix(v, "/") || strings.Contains(v, "\n") {
		return nil
	}
	word := strings.Fields(v + " ")[0]
	if strings.Contains(v, " ") && len(strings.Fields(v)) > 0 {
		// arguments are being typed: keep the popup only for an exact command
		for _, c := range slashCmds {
			if c.name == word {
				return []cmdDef{c}
			}
		}
		return nil
	}
	var out []cmdDef
	for _, c := range slashCmds {
		if strings.HasPrefix(c.name, word) {
			out = append(out, c)
		}
	}
	return out
}

func (m *Model) popupView() string {
	items := m.popupItems()
	if len(items) == 0 {
		return ""
	}
	dim := lipgloss.NewStyle().Foreground(m.th.Dim)
	acc := lipgloss.NewStyle().Foreground(m.th.Accent)
	var rows []string
	for i, c := range items {
		line := fmt.Sprintf("  %-10s %s", c.name, c.desc)
		if i == m.popSel {
			rows = append(rows, acc.Render(line))
		} else {
			rows = append(rows, dim.Render(line))
		}
	}
	return strings.Join(rows, "\n")
}

// runCommand executes a slash command.
func (m *Model) runCommand(name, arg string) tea.Cmd {
	m.ta.Reset()
	m.popSel = 0
	switch name {
	case "/quit":
		return tea.Quit
	case "/clear":
		m.clearView()
	case "/new":
		m.clearView()
		m.sid = ""
		m.setBusy(false)
		return m.createSession()
	case "/sessions":
		c := m.c
		return func() tea.Msg {
			s, err := c.ListSessions(bg())
			if err != nil {
				return errMsg{err}
			}
			return sessionsMsg{s}
		}
	case "/model":
		return m.openModels(strings.TrimSpace(arg))
	}
	return nil
}

func (m *Model) clearView() {
	m.blocks = nil
	m.byPart = map[string]*Block{}
	m.vp.SetContent("")
}

func (m *Model) openModels(arg string) tea.Cmd {
	var items []pickItem
	for _, pr := range m.provs.Providers {
		for _, md := range pr.Models {
			ref := pr.ID + "/" + md.ID
			if arg != "" && (arg == ref || arg == md.ID) {
				m.model = oc.ModelRef{ProviderID: pr.ID, ModelID: md.ID}
				m.layout()
				return nil
			}
			tag := ""
			if md.Cost.Input == 0 {
				tag = "  (free)"
			}
			items = append(items, pickItem{ref + tag, ref})
		}
	}
	if arg != "" {
		m.add(&Block{Kind: KError, Text: "unknown model: " + arg})
		return nil
	}
	sort.Slice(items, func(i, j int) bool {
		fi, fj := strings.HasSuffix(items[i].label, "(free)"), strings.HasSuffix(items[j].label, "(free)")
		if fi != fj {
			return fi
		}
		return items[i].label < items[j].label
	})
	m.pick = &picker{title: "Choose a model", kind: "model", items: items}
	m.layout()
	return nil
}

func (m *Model) openSessions(ss []oc.Session) {
	sort.Slice(ss, func(i, j int) bool { return ss[i].Time.Updated > ss[j].Time.Updated })
	var items []pickItem
	for _, s := range ss {
		t := time.UnixMilli(s.Time.Updated).Format("Jan 2 15:04")
		title := s.Title
		if title == "" {
			title = s.ID
		}
		items = append(items, pickItem{fmt.Sprintf("%-12s %s", t, title), s.ID})
	}
	if len(items) == 0 {
		m.add(&Block{Kind: KError, Text: "no sessions yet"})
		return
	}
	m.pick = &picker{title: "Resume a session", kind: "session", items: items}
	m.layout()
}

func (m *Model) pickerView() string {
	p := m.pick
	acc := lipgloss.NewStyle().Foreground(m.th.Accent)
	dim := lipgloss.NewStyle().Foreground(m.th.Dim)
	rows := []string{lipgloss.NewStyle().Bold(true).Render(p.title), ""}
	// show a window of up to 10 rows around the selection
	start := 0
	if p.sel >= 10 {
		start = p.sel - 9
	}
	end := start + 10
	if end > len(p.items) {
		end = len(p.items)
	}
	for i := start; i < end; i++ {
		l := p.items[i].label
		if lipgloss.Width(l) > m.w-8 {
			l = truncate(l, m.w-8)
		}
		if i == p.sel {
			rows = append(rows, acc.Render("> "+l))
		} else {
			rows = append(rows, "  "+l)
		}
	}
	rows = append(rows, "", dim.Render("up/down to move, enter to choose, esc to cancel"))
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(m.th.Accent).Width(m.w - 2).Render(strings.Join(rows, "\n"))
}

func (m *Model) pickerKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := m.pick
	switch k.String() {
	case "up", "k":
		if p.sel > 0 {
			p.sel--
		}
	case "down", "j":
		if p.sel < len(p.items)-1 {
			p.sel++
		}
	case "esc":
		m.pick = nil
		m.layout()
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		it := p.items[p.sel]
		m.pick = nil
		m.layout()
		if p.kind == "model" {
			i := strings.Index(it.value, "/")
			m.model = oc.ModelRef{ProviderID: it.value[:i], ModelID: it.value[i+1:]}
			m.layout()
			return m, nil
		}
		return m, m.resume(it.value)
	}
	return m, nil
}

func (m *Model) resume(sid string) tea.Cmd {
	c := m.c
	return func() tea.Msg {
		ss, err := c.ListSessions(bg())
		var s oc.Session
		for _, x := range ss {
			if x.ID == sid {
				s = x
			}
		}
		if s.ID == "" {
			s.ID = sid
		}
		if err != nil {
			s.ID = sid
		}
		msgs, err := c.Messages(bg(), sid)
		if err != nil {
			return errMsg{err}
		}
		return resumeMsg{s, msgs}
	}
}

// loadHistory rebuilds the transcript from a stored session.
func (m *Model) loadHistory(r resumeMsg) {
	m.clearView()
	m.sid = r.s.ID
	m.perm = nil
	m.roles = map[string]string{}
	for _, mm := range r.msgs {
		m.roles[mm.Info.ID] = mm.Info.Role
	}
	for _, mm := range r.msgs {
		for _, p := range mm.Parts {
			if mm.Info.Role == "user" {
				if p.Type == "text" && p.Text != "" {
					m.add(&Block{Kind: KUser, Text: p.Text})
				}
				continue
			}
			p.SessionID = m.sid
			m.onPart(oc.PartUpdated{SessionID: m.sid, Part: p})
		}
	}
}

func (m *Model) shortcutsView() string {
	dim := lipgloss.NewStyle().Foreground(m.th.Dim)
	rows := []string{
		"Enter send   Alt+Enter or Ctrl+J newline   Esc interrupt   Ctrl+C quit",
		"Ctrl+O toggle tool output   Ctrl+L clear   Up/Down input history",
		"/ for commands: /new /sessions /model /clear /quit",
	}
	for i := range rows {
		rows[i] = dim.Render(rows[i])
	}
	return strings.Join(rows, "\n")
}
