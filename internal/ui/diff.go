package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type diffLine struct {
	op   byte // ' ', '+', '-'
	num  int
	text string
}

// parseUnified reads a unified diff and numbers each line.
func parseUnified(d string) []diffLine {
	var out []diffLine
	oldN, newN := 0, 0
	for _, l := range strings.Split(strings.TrimRight(d, "\n"), "\n") {
		switch {
		case strings.HasPrefix(l, "@@"):
			oldN, newN = hunkStart(l)
		case strings.HasPrefix(l, "Index:"), strings.HasPrefix(l, "==="),
			strings.HasPrefix(l, "--- "), strings.HasPrefix(l, "+++ "), strings.HasPrefix(l, "diff "):
		case strings.HasPrefix(l, "\\"):
		case strings.HasPrefix(l, "+"):
			out = append(out, diffLine{'+', newN, l[1:]})
			newN++
		case strings.HasPrefix(l, "-"):
			out = append(out, diffLine{'-', oldN, l[1:]})
			oldN++
		default:
			t := l
			if strings.HasPrefix(t, " ") {
				t = t[1:]
			}
			out = append(out, diffLine{' ', newN, t})
			oldN++
			newN++
		}
	}
	return out
}

func hunkStart(h string) (int, int) {
	var a, b int
	f := strings.Fields(h)
	for _, x := range f {
		if len(x) > 1 && (x[0] == '-' || x[0] == '+') {
			n := strings.SplitN(x[1:], ",", 2)[0]
			v, err := strconv.Atoi(n)
			if err != nil {
				continue
			}
			if x[0] == '-' {
				a = v
			} else {
				b = v
			}
		}
	}
	return a, b
}

// synthDiff builds diff lines from edit/write input when the server sent no diff.
func synthDiff(in map[string]any) []diffLine {
	var out []diffLine
	if c := str(in, "content"); c != "" {
		for i, l := range strings.Split(strings.TrimRight(c, "\n"), "\n") {
			out = append(out, diffLine{'+', i + 1, l})
		}
		return out
	}
	o, n := str(in, "oldString"), str(in, "newString")
	for i, l := range strings.Split(strings.TrimRight(o, "\n"), "\n") {
		if o != "" {
			out = append(out, diffLine{'-', i + 1, l})
		}
	}
	for i, l := range strings.Split(strings.TrimRight(n, "\n"), "\n") {
		if n != "" {
			out = append(out, diffLine{'+', i + 1, l})
		}
	}
	return out
}

func diffCounts(ls []diffLine) (add, del int) {
	for _, l := range ls {
		switch l.op {
		case '+':
			add++
		case '-':
			del++
		}
	}
	return
}

// renderDiffLines draws numbered lines with green and red tints, at most max lines.
func renderDiffLines(th Theme, ls []diffLine, w, max int) []string {
	dim := lipgloss.NewStyle().Foreground(th.Dim)
	add := lipgloss.NewStyle().Background(th.AddBg).Foreground(th.Fg)
	del := lipgloss.NewStyle().Background(th.DelBg).Foreground(th.Fg)
	var out []string
	cut := 0
	for i, l := range ls {
		if max > 0 && i >= max {
			cut = len(ls) - i
			break
		}
		text := strings.ReplaceAll(l.text, "\t", "    ")
		gut := fmt.Sprintf("%4d ", l.num)
		sign := string(l.op)
		room := w - 2 - len(gut) - 2
		if room < 4 {
			room = 4
		}
		if lipgloss.Width(text) > room {
			text = truncate(text, room)
		}
		row := gut + sign + " " + text
		pad := w - 2 - lipgloss.Width(row)
		if pad < 0 {
			pad = 0
		}
		row += strings.Repeat(" ", pad)
		switch l.op {
		case '+':
			out = append(out, "  "+add.Render(row))
		case '-':
			out = append(out, "  "+del.Render(row))
		default:
			out = append(out, "  "+dim.Render(row))
		}
	}
	if cut > 0 {
		out = append(out, dim.Render(fmt.Sprintf("  ... +%d lines", cut)))
	}
	return out
}

func truncate(s string, w int) string {
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r)) > w-1 {
		r = r[:len(r)-1]
	}
	return string(r) + "~"
}

func isEditTool(t string) bool {
	return t == "edit" || t == "write" || t == "patch" || t == "multiedit"
}

func (b *Block) diffLines() []diffLine {
	if d, ok := b.Meta["diff"].(string); ok && d != "" {
		return parseUnified(d)
	}
	return synthDiff(b.Input)
}

func renderDiffBlock(th Theme, b *Block, w int, expanded bool) string {
	verb, subj := toolHeader(b)
	dim := lipgloss.NewStyle().Foreground(th.Dim)
	if b.Status != "completed" {
		l := "* " + verb + " " + subj
		if b.Status == "error" {
			return lipgloss.NewStyle().Foreground(th.Err).Render("!") + l[1:] + "\n" + dim.Render("    "+b.Output)
		}
		return dim.Render("*") + l[1:]
	}
	ls := b.diffLines()
	a, d := diffCounts(ls)
	head := fmt.Sprintf("* %s %s (+%d -%d)", verb, subj, a, d)
	max := 40
	if expanded {
		max = 0
	}
	rows := append([]string{dim.Render("*") + head[1:]}, renderDiffLines(th, ls, w, max)...)
	return strings.Join(rows, "\n")
}
