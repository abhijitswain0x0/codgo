package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var nowFn = time.Now

const maxInputLines = 8

func (m *Model) inputBox() string {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(m.th.Border).Width(m.w - 2).Render(m.ta.View())
}

func (m *Model) busyLine() string {
	if !m.busy {
		return ""
	}
	return m.sp.View() + " " + lipgloss.NewStyle().Foreground(m.th.Dim).
		Render("Working ("+m.elapsed()+" - esc to interrupt)")
}

func (m *Model) hintLine() string {
	dim := lipgloss.NewStyle().Foreground(m.th.Dim)
	left, right := "? for shortcuts", m.modelLabel()
	gap := m.w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return dim.Render(left + strings.Repeat(" ", gap) + right)
}

func (m *Model) statusLine() string {
	state := "idle"
	if m.busy {
		state = "busy"
	}
	conn := ""
	if !m.streamUp {
		conn = "  reconnecting"
	}
	s := m.modelLabel() + "  " + m.cwd + "  " + m.shortSID() + "  " + state + conn
	return lipgloss.NewStyle().Foreground(m.th.Dim).MaxWidth(m.w).Render(s)
}

func (m *Model) bottom() string {
	parts := []string{}
	if m.busy {
		parts = append(parts, m.busyLine())
	}
	if len(m.perm) > 0 {
		parts = append(parts, m.modal(), m.statusLine())
	} else if m.pick != nil {
		parts = append(parts, m.pickerView(), m.statusLine())
	} else {
		if pv := m.popupView(); pv != "" {
			parts = append(parts, pv)
		}
		if m.help {
			parts = append(parts, m.shortcutsView())
		}
		parts = append(parts, m.inputBox(), m.hintLine(), m.statusLine())
	}
	return strings.Join(parts, "\n")
}

func (m *Model) layout() {
	if m.w == 0 {
		return
	}
	n := m.ta.LineCount()
	if n < 1 {
		n = 1
	}
	if n > maxInputLines {
		n = maxInputLines
	}
	m.ta.SetHeight(n)
	m.ta.SetWidth(m.w - 4)
	bh := lipgloss.Height(m.bottom())
	vh := m.h - bh
	if vh < 1 {
		vh = 1
	}
	if m.vp.Width == 0 {
		m.vp = newVP(m.w, vh)
	}
	m.vp.Width, m.vp.Height = m.w, vh
	m.refresh(true)
}

func (m *Model) refresh(stick bool) {
	if m.w == 0 {
		return
	}
	parts := make([]string, 0, len(m.blocks))
	for _, b := range m.blocks {
		parts = append(parts, b.render(m.th, m.w, m.expanded))
	}
	m.vp.SetContent(strings.Join(parts, "\n\n"))
	if stick {
		m.vp.GotoBottom()
	}
}

func (m *Model) View() string {
	if m.w == 0 {
		return ""
	}
	return m.vp.View() + "\n" + m.bottom()
}
