package ui

import (
	"testing"

	"github.com/abhijitswain0x0/codgo/internal/oc"
	tea "github.com/charmbracelet/bubbletea"
)

func TestView(t *testing.T) {
	m := New(oc.New("http://x", "/p"), "/p", DefaultTheme(), "")
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 12})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("abc")})
	t.Log("\n" + m.View())
}
