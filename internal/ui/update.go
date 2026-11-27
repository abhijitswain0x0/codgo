package ui

import (
	"encoding/json"
	"strings"

	"github.com/abhijitswain0x0/codgo/internal/oc"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Messages that come from our own commands.
type (
	sessionMsg   struct{ s oc.Session }
	providersMsg struct{ p oc.Providers }
	errMsg       struct{ err error }
	pendingMsg   struct{ p []oc.PermissionAsked }
)

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.createSession(), m.loadProviders())
}

func (m *Model) createSession() tea.Cmd {
	return func() tea.Msg {
		s, err := m.c.CreateSession(bg())
		if err != nil {
			return errMsg{err}
		}
		return sessionMsg{s}
	}
}

func (m *Model) loadProviders() tea.Cmd {
	return func() tea.Msg {
		p, err := m.c.Providers(bg())
		if err != nil {
			return errMsg{err}
		}
		return providersMsg{p}
	}
}

func (m *Model) send(text string) tea.Cmd {
	sid, ref := m.sid, m.model
	return func() tea.Msg {
		if err := m.c.Prompt(bg(), sid, &ref, text); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func pickModel(p oc.Providers) oc.ModelRef {
	if id, ok := p.Default["opencode"]; ok {
		return oc.ModelRef{ProviderID: "opencode", ModelID: id}
	}
	for _, pr := range p.Providers {
		for _, md := range pr.Models {
			if md.Cost.Input == 0 {
				return oc.ModelRef{ProviderID: pr.ID, ModelID: md.ID}
			}
		}
	}
	for k, v := range p.Default {
		return oc.ModelRef{ProviderID: k, ModelID: v}
	}
	return oc.ModelRef{}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	atBottom := !m.ready || m.vp.AtBottom()
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.layout()
		m.ready = true
		atBottom = true
	case tea.KeyMsg:
		if len(m.perm) > 0 {
			return m.permKey(msg)
		}
		if m.pick != nil {
			return m.pickerKey(msg)
		}
		if m.help {
			m.help = false
			m.layout()
		}
		pop := m.popupItems()
		switch msg.String() {
		case "?":
			if m.ta.Value() == "" {
				m.help = true
				m.layout()
				return m, nil
			}
		case "ctrl+l":
			m.clearView()
			return m, nil
		case "esc":
			if len(pop) > 0 {
				m.ta.Reset()
				m.layout()
				return m, nil
			}
			if m.busy && m.sid != "" {
				sid, c := m.sid, m.c
				return m, func() tea.Msg {
					if err := c.Abort(bg(), sid); err != nil {
						return errMsg{err}
					}
					return nil
				}
			}
			return m, nil
		case "tab":
			if len(pop) > 0 {
				m.ta.SetValue(pop[m.popSel].name + " ")
				m.ta.CursorEnd()
				m.layout()
				return m, nil
			}
		case "up", "down":
			if len(pop) > 1 {
				if msg.String() == "up" && m.popSel > 0 {
					m.popSel--
				}
				if msg.String() == "down" && m.popSel < len(pop)-1 {
					m.popSel++
				}
				return m, nil
			}
			if m.historyKey(msg.String()) {
				m.layout()
				return m, nil
			}
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+o":
			m.expanded = !m.expanded
			m.refresh(false)
			return m, nil
		case "enter":
			text := strings.TrimSpace(m.ta.Value())
			if len(pop) > 0 {
				if m.popSel >= len(pop) {
					m.popSel = 0
				}
				arg := ""
				if f := strings.Fields(text); len(f) > 1 {
					arg = strings.Join(f[1:], " ")
				}
				c := m.runCommand(pop[m.popSel].name, arg)
				m.layout()
				return m, c
			}
			if text == "" || m.sid == "" {
				return m, nil
			}
			m.pushHistory(text)
			saveHistory(text)
			m.ta.Reset()
			m.ta.Placeholder = ""
			m.add(&Block{Kind: KUser, Text: text})
			m.setBusy(true)
			atBottom = true
			cmds = append(cmds, m.send(text), m.sp.Tick)
			m.layout()
			m.refresh(atBottom)
			return m, tea.Batch(cmds...)
		}
		var c tea.Cmd
		m.ta, c = m.ta.Update(msg)
		cmds = append(cmds, c)
		m.layout()
	case spinner.TickMsg:
		if m.busy {
			var c tea.Cmd
			m.sp, c = m.sp.Update(msg)
			cmds = append(cmds, c)
		}
	case sessionMsg:
		m.sid = msg.s.ID
		m.roles = map[string]string{}
		cmds = append(cmds, m.loadPending())
	case pendingMsg:
		for _, p := range msg.p {
			m.addPerm(p)
		}
		m.layout()
	case oc.PermissionAsked:
		m.addPerm(msg)
		m.layout()
	case oc.PermissionReplied:
		m.dropPerm(msg.RequestID)
		m.layout()
	case sessionsMsg:
		m.openSessions(msg.s)
	case resumeMsg:
		m.loadHistory(msg)
		m.layout()
		atBottom = true
		cmds = append(cmds, m.loadPending())
	case providersMsg:
		m.provs = msg.p
		if m.model.ModelID == "" {
			m.model = pickModel(msg.p)
		}
	case errMsg:
		m.add(&Block{Kind: KError, Text: msg.err.Error()})
		m.setBusy(false)
	case oc.SessionError:
		if msg.SessionID == m.sid {
			m.add(errorBlock(msg.Error))
			m.setBusy(false)
		}
	case oc.StreamState:
		reconnected := msg.Up && !m.streamUp && m.sawDown
		m.streamUp = msg.Up
		if !msg.Up {
			m.sawDown = true
		}
		if reconnected && m.sid != "" {
			cmds = append(cmds, m.loadPending())
		}
	case oc.MessageUpdated:
		if msg.SessionID == m.sid {
			m.roles[msg.Info.ID] = msg.Info.Role
		}
	case oc.PartUpdated:
		m.onPart(msg)
	case oc.PartDelta:
		m.onDelta(msg)
	case oc.SessionStatus:
		if msg.SessionID == m.sid {
			on := msg.Status.Type == "busy"
			if on && !m.busy {
				cmds = append(cmds, m.sp.Tick)
			}
			m.setBusy(on)
		}
	case oc.SessionIdle:
		if msg.SessionID == m.sid {
			m.setBusy(false)
		}
	}
	m.refresh(atBottom)
	return m, tea.Batch(cmds...)
}

func (m *Model) setBusy(on bool) {
	if on && !m.busy {
		m.busySince = nowFn()
	}
	m.busy = on
	m.layout()
}

func (m *Model) onPart(e oc.PartUpdated) {
	if e.SessionID != m.sid {
		return
	}
	p := e.Part
	if m.roles[p.MessageID] == "user" {
		return
	}
	switch p.Type {
	case "tool":
		b := m.byPart[p.ID]
		if b == nil {
			k := KTool
			if isEditTool(p.Tool) {
				k = KDiff
			}
			b = m.add(&Block{Kind: k, PartID: p.ID, Tool: p.Tool})
			m.byPart[p.ID] = b
		}
		if st := p.State; st != nil {
			b.Status = st.Status
			if st.Metadata != nil {
				b.Meta = st.Metadata
			}
			if len(st.Input) > 0 {
				b.Input = st.Input
			}
			switch {
			case st.Error != "":
				b.Output = st.Error
			case st.Output != "":
				b.Output = st.Output
			default:
				if o, ok := st.Metadata["output"].(string); ok {
					b.Output = o
				}
			}
		}
		b.touch()
	case "text":
		b := m.byPart[p.ID]
		if b == nil {
			b = m.add(&Block{Kind: KText, PartID: p.ID})
			m.byPart[p.ID] = b
		}
		b.Text = strings.TrimLeft(p.Text, "\n")
		b.touch()
	}
}

func (m *Model) onDelta(e oc.PartDelta) {
	if e.SessionID != m.sid || e.Field != "text" {
		return
	}
	b := m.byPart[e.PartID]
	if b == nil {
		return // reasoning and unknown parts are not shown
	}
	b.Text += e.Delta
	b.touch()
}

// errorBlock builds a short message plus details from a session.error payload.
func errorBlock(raw json.RawMessage) *Block {
	var e struct {
		Name string `json:"name"`
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &e)
	if e.Name == "MessageAbortedError" {
		return &Block{Kind: KError, Text: "Interrupted", dirty: true}
	}
	msg := e.Data.Message
	if msg == "" {
		msg = e.Name
	}
	if msg == "" {
		msg = "the agent hit an error"
	}
	short := msg
	if i := strings.IndexByte(short, '\n'); i >= 0 {
		short = short[:i]
	}
	if len(short) > 160 {
		short = short[:160] + "..."
	}
	return &Block{Kind: KError, Text: short, Detail: string(raw), dirty: true}
}

func (m *Model) loadPending() tea.Cmd {
	return func() tea.Msg {
		p, err := m.c.Pending(bg())
		if err != nil {
			return nil
		}
		return pendingMsg{p}
	}
}

func (m *Model) addPerm(p oc.PermissionAsked) {
	if p.SessionID != m.sid {
		return
	}
	for _, q := range m.perm {
		if q.ID == p.ID {
			return
		}
	}
	m.perm = append(m.perm, p)
	m.permSel = 0
}

func (m *Model) dropPerm(id string) {
	for i, q := range m.perm {
		if q.ID == id {
			m.perm = append(m.perm[:i], m.perm[i+1:]...)
			m.permSel = 0
			return
		}
	}
}

// permKey handles keys while the approval modal is showing.
func (m *Model) permKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	choice := -1
	switch k.String() {
	case "y", "1":
		choice = 0
	case "a", "2":
		choice = 1
	case "esc", "n", "3":
		choice = 2
	case "up", "k":
		if m.permSel > 0 {
			m.permSel--
		}
	case "down", "j":
		if m.permSel < 2 {
			m.permSel++
		}
	case "enter":
		choice = m.permSel
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+o":
		m.expanded = !m.expanded
		m.refresh(false)
	}
	if choice < 0 {
		return m, nil
	}
	p := m.perm[0]
	reply := [...]string{"once", "always", "reject"}[choice]
	m.dropPerm(p.ID)
	m.layout()
	if choice == 2 {
		m.ta.Placeholder = "Tell the agent what to do differently"
		m.ta.Focus()
	}
	c := m.c
	return m, func() tea.Msg {
		if err := c.Reply(bg(), p.ID, reply, ""); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func (m *Model) pushHistory(t string) {
	if n := len(m.hist); n == 0 || m.hist[n-1] != t {
		m.hist = append(m.hist, t)
	}
	m.histIdx = len(m.hist)
}

// historyKey recalls earlier input on up/down when the cursor is on the first or
// last line. It reports whether it consumed the key.
func (m *Model) historyKey(k string) bool {
	if len(m.hist) == 0 {
		return false
	}
	if k == "up" && m.ta.Line() == 0 && m.histIdx > 0 {
		m.histIdx--
		m.ta.SetValue(m.hist[m.histIdx])
		return true
	}
	if k == "down" && m.ta.Line() == m.ta.LineCount()-1 && m.histIdx < len(m.hist) {
		m.histIdx++
		if m.histIdx == len(m.hist) {
			m.ta.Reset()
		} else {
			m.ta.SetValue(m.hist[m.histIdx])
		}
		return true
	}
	return false
}
