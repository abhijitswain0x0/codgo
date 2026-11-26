package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Theme holds every color codgo uses. Nothing else in the package names a color.
type Theme struct {
	Fg, Dim, Accent, Err, Ok lipgloss.TerminalColor
	UserBg, AddBg, DelBg     lipgloss.TerminalColor
	AddFg, DelFg             lipgloss.TerminalColor
	Border                   lipgloss.TerminalColor
}

// DefaultTheme is a dark, low-chrome palette.
func DefaultTheme() Theme {
	return Theme{
		Fg:     lipgloss.Color("252"),
		Dim:    lipgloss.Color("244"),
		Accent: lipgloss.Color("75"),
		Err:    lipgloss.Color("203"),
		Ok:     lipgloss.Color("114"),
		UserBg: lipgloss.Color("236"),
		AddBg:  lipgloss.Color("22"),
		DelBg:  lipgloss.Color("52"),
		AddFg:  lipgloss.Color("114"),
		DelFg:  lipgloss.Color("210"),
		Border: lipgloss.Color("240"),
	}
}

// Theme16 uses only the 16 basic ANSI colors.
func Theme16() Theme {
	return Theme{
		Fg:     lipgloss.Color("7"),
		Dim:    lipgloss.Color("8"),
		Accent: lipgloss.Color("6"),
		Err:    lipgloss.Color("1"),
		Ok:     lipgloss.Color("2"),
		UserBg: lipgloss.Color("0"),
		AddBg:  lipgloss.Color("2"),
		DelBg:  lipgloss.Color("1"),
		AddFg:  lipgloss.Color("2"),
		DelFg:  lipgloss.Color("1"),
		Border: lipgloss.Color("8"),
	}
}

// PickTheme chooses the palette for the terminal's color support.
func PickTheme() Theme {
	switch lipgloss.ColorProfile() {
	case termenv.Ascii:
		n := lipgloss.NoColor{}
		return Theme{Fg: n, Dim: n, Accent: n, Err: n, Ok: n, UserBg: n, AddBg: n, DelBg: n, AddFg: n, DelFg: n, Border: n}
	case termenv.ANSI:
		return Theme16()
	}
	return DefaultTheme()
}
