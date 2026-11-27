package ui

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const maxHistory = 200

func historyPath() string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(h, ".local", "state")
	}
	return filepath.Join(dir, "codgo", "history")
}

func loadHistory() []string {
	p := historyPath()
	if p == "" {
		return nil
	}
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		out = append(out, strings.ReplaceAll(sc.Text(), `\n`, "\n"))
	}
	if len(out) > maxHistory {
		out = out[len(out)-maxHistory:]
	}
	return out
}

// saveHistory appends one entry. Failures are ignored: history is a convenience.
func saveHistory(t string) {
	p := historyPath()
	if p == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(strings.ReplaceAll(t, "\n", `\n`) + "\n")
}
