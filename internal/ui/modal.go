package ui

import (
	"fmt"
	"strings"

	"github.com/abhijitswain0x0/codgo/internal/oc"
	"github.com/charmbracelet/lipgloss"
)

var permOptions = []string{
	"Yes, proceed (y)",
	"Yes, and do not ask again for this (a)",
	"No, and tell the agent what to do differently (esc)",
}

func permTitle(p *oc.PermissionAsked) string {
	switch p.Permission {
	case "bash":
		return "Would you like to run the following command?"
	case "edit", "write", "patch", "multiedit":
		return "Would you like to make the following edits?"
	}
	return "Would you like to allow the agent to use " + p.Permission + "?"
}

func permPreview(th Theme, p *oc.PermissionAsked, w int) []string {
	const maxRows = 12
	if cmd, ok := p.Metadata["command"].(string); ok {
		ls := strings.Split(wrap(cmd, w-4), "\n")
		if len(ls) > maxRows {
			ls = append(ls[:maxRows], fmt.Sprintf("... +%d lines", len(ls)-maxRows))
		}
		for i := range ls {
			ls[i] = "  $ " + ls[i]
			if i > 0 {
				ls[i] = "    " + strings.TrimPrefix(ls[i], "  $ ")
			}
		}
		return ls
	}
	if d, ok := p.Metadata["diff"].(string); ok && d != "" {
		rows := []string{}
		if f, ok := p.Metadata["filepath"].(string); ok {
			rows = append(rows, "  "+f)
		}
		return append(rows, renderDiffLines(th, parseUnified(d), w-2, maxRows)...)
	}
	return []string{"  " + strings.Join(p.Patterns, ", ")}
}

func (m *Model) modal() string {
	p := m.perm[0]
	w := m.w - 4
	accent := lipgloss.NewStyle().Foreground(m.th.Accent)
	dim := lipgloss.NewStyle().Foreground(m.th.Dim)
	rows := []string{lipgloss.NewStyle().Bold(true).Render(permTitle(&p)), ""}
	rows = append(rows, permPreview(m.th, &p, w)...)
	rows = append(rows, "")
	for i, o := range permOptions {
		line := fmt.Sprintf("%d. %s", i+1, o)
		if i == m.permSel {
			rows = append(rows, accent.Render("> "+line))
		} else {
			rows = append(rows, "  "+line)
		}
	}
	rows = append(rows, "", dim.Render("up/down to move, enter to choose"))
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(m.th.Accent).Width(m.w - 2).Render(strings.Join(rows, "\n"))
}
