package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type Kind int

const (
	KUser Kind = iota
	KText
	KTool
	KDiff
	KError
)

// Block is one entry in the transcript. Rendering is cached per width.
type Block struct {
	Kind   Kind
	PartID string // opencode part id, "" for local blocks
	Text   string

	// tool blocks
	Tool   string
	Status string // pending, running, completed, error
	Input  map[string]any
	Output string
	Meta   map[string]any

	// error blocks: Text is the short message, Detail the full text
	Detail string

	cache   string
	cacheW  int
	dirty   bool
	cacheEx bool
}

func (b *Block) touch() { b.dirty = true }

func (b *Block) render(th Theme, w int, expanded bool) string {
	if !b.dirty && b.cacheW == w && b.cacheEx == expanded && b.cache != "" {
		return b.cache
	}
	var s string
	switch b.Kind {
	case KUser:
		s = renderUser(th, b.Text, w)
	case KDiff:
		s = renderDiffBlock(th, b, w, expanded)
	case KTool:
		s = renderTool(th, b, w, expanded)
	case KError:
		s = renderError(th, b, w, expanded)
	default:
		s = renderText(th, b.Text, w)
	}
	b.cache, b.cacheW, b.cacheEx, b.dirty = s, w, expanded, false
	return s
}

func wrap(s string, w int) string {
	if w < 8 {
		w = 8
	}
	return ansi.Wrap(s, w, "")
}

// prefixed wraps text and puts first on the first line, rest on the others.
func prefixed(first, rest, text string, w int) []string {
	pw := lipgloss.Width(first)
	lines := strings.Split(wrap(text, w-pw), "\n")
	for i := range lines {
		if i == 0 {
			lines[i] = first + lines[i]
		} else {
			lines[i] = rest + lines[i]
		}
	}
	return lines
}

func renderText(th Theme, text string, w int) string {
	lines := prefixed("* ", "  ", text, w)
	lines[0] = lipgloss.NewStyle().Foreground(th.Dim).Render("*") + lines[0][1:]
	return strings.Join(lines, "\n")
}

func renderUser(th Theme, text string, w int) string {
	st := lipgloss.NewStyle().Background(th.UserBg).Foreground(th.Fg)
	lines := prefixed("> ", "  ", text, w)
	for i := range lines {
		pad := w - lipgloss.Width(lines[i])
		if pad < 0 {
			pad = 0
		}
		lines[i] = st.Render(lines[i] + strings.Repeat(" ", pad))
	}
	lines[0] = strings.Replace(lines[0], ">", lipgloss.NewStyle().Background(th.UserBg).Foreground(th.Accent).Render(">"), 1)
	return strings.Join(lines, "\n")
}

const previewLines = 5

func str(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// toolHeader returns the verb and subject for a tool call.
func toolHeader(b *Block) (verb, subject string) {
	done := b.Status == "completed"
	pick := func(now, past string) string {
		if done {
			return past
		}
		return now
	}
	switch b.Tool {
	case "bash":
		return pick("Running", "Ran"), str(b.Input, "command")
	case "read":
		return pick("Reading", "Read"), str(b.Input, "filePath", "path")
	case "write":
		return pick("Writing", "Wrote"), str(b.Input, "filePath", "path")
	case "edit", "patch", "multiedit":
		return pick("Editing", "Edited"), str(b.Input, "filePath", "path")
	case "grep", "glob":
		return pick("Searching", "Searched"), str(b.Input, "pattern")
	case "list":
		return pick("Listing", "Listed"), str(b.Input, "path")
	case "webfetch":
		return pick("Fetching", "Fetched"), str(b.Input, "url")
	case "task":
		return pick("Delegating", "Delegated"), str(b.Input, "description")
	}
	return pick("Using", "Used") + " " + b.Tool, str(b.Input, "description", "filePath", "pattern", "command")
}

func renderTool(th Theme, b *Block, w int, expanded bool) string {
	verb, subj := toolHeader(b)
	dim := lipgloss.NewStyle().Foreground(th.Dim)
	head := verb
	if subj != "" {
		head += " " + subj
	}
	lines := prefixed("* ", "  ", head, w)
	lines[0] = dim.Render("*") + lines[0][1:]
	if b.Status == "error" {
		lines[0] = lipgloss.NewStyle().Foreground(th.Err).Render("!") + lines[0][1:]
	}
	out := strings.TrimRight(b.Output, "\n ")
	if out == "" {
		return strings.Join(lines, "\n")
	}
	body := strings.Split(wrap(out, w-4), "\n")
	extra := 0
	if !expanded && len(body) > previewLines {
		extra = len(body) - previewLines
		body = body[:previewLines]
	}
	for i, l := range body {
		conn := "    "
		if i == 0 {
			conn = "  L "
		}
		lines = append(lines, dim.Render(conn+l))
	}
	if extra > 0 {
		lines = append(lines, dim.Render(fmt.Sprintf("    ... +%d lines", extra)))
	}
	return strings.Join(lines, "\n")
}

func renderError(th Theme, b *Block, w int, expanded bool) string {
	red := lipgloss.NewStyle().Foreground(th.Err)
	lines := prefixed("! ", "  ", b.Text, w)
	lines[0] = red.Render("!") + lines[0][1:]
	if b.Detail != "" && b.Detail != b.Text {
		if expanded {
			for _, l := range strings.Split(wrap(b.Detail, w-4), "\n") {
				lines = append(lines, lipgloss.NewStyle().Foreground(th.Dim).Render("    "+l))
			}
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(th.Dim).Render("    details hidden (ctrl+o)"))
		}
	}
	return strings.Join(lines, "\n")
}
