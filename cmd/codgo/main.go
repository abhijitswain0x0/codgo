package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/abhijitswain0x0/codgo/internal/oc"
	"github.com/abhijitswain0x0/codgo/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	url := flag.String("url", os.Getenv("CODGO_URL"), "opencode server URL (default: start one)")
	model := flag.String("model", os.Getenv("CODGO_MODEL"), "model as provider/model")
	flag.Parse()

	cwd, _ := os.Getwd()
	srv, base, err := ensureServer(*url, cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "codgo:", err)
		os.Exit(1)
	}
	defer srv.Stop()

	c := oc.New(base, cwd)
	m := ui.New(c, cwd, ui.PickTheme(), *model)
	p := tea.NewProgram(m, tea.WithAltScreen())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Stream(ctx, func(v any) { p.Send(v) })

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "codgo:", err)
		srv.Stop()
		os.Exit(1)
	}
}
