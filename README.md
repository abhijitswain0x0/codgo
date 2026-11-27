# codgo

A terminal UI for [opencode](https://opencode.ai), styled after the Codex CLI.
opencode does the agent work. codgo only renders and controls it.

    codgo (Go TUI) <-- HTTP + SSE --> opencode serve <--> opencode free (Zen) models

Plain text only (no markdown rendering). One static Go binary, no CGO, five direct
dependencies (Bubble Tea, Lip Gloss, Bubbles, x/ansi for word wrap, termenv for color detection). Idle CPU is near
zero: the UI wakes only on key presses and server events, and the spinner ticks only
while the agent is working.

## Setup

1. Install opencode and make sure `opencode` is on your PATH.
2. Get codgo: download the `codgo-linux-amd64` artifact from the latest
   GitHub Actions run (Actions tab, `build` workflow), or build it:

        make build        # or: CGO_ENABLED=0 go build ./cmd/codgo

3. Run `./codgo` in the project directory you want the agent to work in.

codgo starts `opencode serve` on a free local port if none is running, and stops it
when you quit. If it finds a server on `127.0.0.1:4096`, or you pass `--url`, it uses that
and leaves it running. The server codgo starts is configured so edits and shell commands
ask first (that is what drives the approval prompt). Set `OPENCODE_CONFIG_CONTENT`
yourself to override this.

Flags and environment:

| Option | Meaning |
| --- | --- |
| `--url`, `CODGO_URL` | use an already running opencode server |
| `--model`, `CODGO_MODEL` | model as `provider/model`, for example `opencode/big-pickle` |
| `NO_COLOR` | plain output; 16-color terminals use a 16-color theme automatically |

## Keys

| Key | Action |
| --- | --- |
| Enter | send |
| Alt+Enter, Ctrl+J | newline (Shift+Enter too, on terminals that report it) |
| Esc | interrupt the agent (aborts the session turn); closes popups |
| Ctrl+C | quit |
| Ctrl+O | show or hide full tool output and error details |
| Up / Down | input history (kept in `~/.local/state/codgo/history`) |
| Ctrl+L | clear the transcript view |
| ? | shortcut help (on an empty input) |

Approval prompt: `y` yes once, `a` yes and do not ask again, `Esc` no. After "no" the input
gets focus so you can tell the agent what to do differently. The next message you send
is that correction. Up/Down and Enter also work in the prompt and in pickers.

## Slash commands

Type `/` for the list.

- `/new` start a new session
- `/sessions` pick and resume a session
- `/model` list models and switch (or `/model provider/model`)
- `/clear` clear the transcript view
- `/quit` exit

## Layout

    cmd/codgo/       main, opencode server start/stop
    internal/oc/     HTTP client and SSE event decoding
    internal/ui/     Bubble Tea model, update, view, blocks, diff, modal, theme
    docs/events.md   the real opencode event shapes codgo was designed against

## Notes

- Colors live in `internal/ui/theme.go`.
- If the event stream drops, codgo shows `reconnecting` in the status line and retries
  with backoff. Server errors show as `!` blocks; nothing crashes the UI.
- Verified against opencode 1.19.6.
