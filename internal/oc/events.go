package oc

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Event is one raw SSE frame.
type Event struct {
	Type  string          `json:"type"`
	Props json.RawMessage `json:"properties"`
}

type ToolState struct {
	Status   string         `json:"status"`
	Input    map[string]any `json:"input"`
	Output   string         `json:"output"`
	Title    string         `json:"title"`
	Error    string         `json:"error"`
	Metadata map[string]any `json:"metadata"`
}

type Part struct {
	ID        string     `json:"id"`
	MessageID string     `json:"messageID"`
	SessionID string     `json:"sessionID"`
	Type      string     `json:"type"`
	Text      string     `json:"text"`
	Tool      string     `json:"tool"`
	CallID    string     `json:"callID"`
	State     *ToolState `json:"state"`
}

type MessageInfo struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	SessionID string `json:"sessionID"`
}

// Typed events. These are the tea.Msg types the UI receives.
type PartUpdated struct {
	SessionID string `json:"sessionID"`
	Part      Part   `json:"part"`
}
type PartDelta struct {
	SessionID string `json:"sessionID"`
	MessageID string `json:"messageID"`
	PartID    string `json:"partID"`
	Field     string `json:"field"`
	Delta     string `json:"delta"`
}
type MessageUpdated struct {
	SessionID string      `json:"sessionID"`
	Info      MessageInfo `json:"info"`
}
type PermissionAsked struct {
	ID         string         `json:"id"`
	SessionID  string         `json:"sessionID"`
	Permission string         `json:"permission"`
	Patterns   []string       `json:"patterns"`
	Metadata   map[string]any `json:"metadata"`
	Always     []string       `json:"always"`
	Tool       *struct {
		MessageID string `json:"messageID"`
		CallID    string `json:"callID"`
	} `json:"tool"`
}
type PermissionReplied struct {
	SessionID string `json:"sessionID"`
	RequestID string `json:"requestID"`
	Reply     string `json:"reply"`
}
type SessionStatus struct {
	SessionID string `json:"sessionID"`
	Status    struct {
		Type string `json:"type"`
	} `json:"status"`
}
type SessionIdle struct {
	SessionID string `json:"sessionID"`
}
type SessionError struct {
	SessionID string          `json:"sessionID"`
	Error     json.RawMessage `json:"error"`
}

// StreamState reports SSE connection changes.
type StreamState struct {
	Up  bool
	Err error
}

// Decode turns a raw event into a typed value, or nil for events codgo ignores.
func Decode(e Event) any {
	var v any
	switch e.Type {
	case "message.part.updated":
		v = &PartUpdated{}
	case "message.part.delta":
		v = &PartDelta{}
	case "message.updated":
		v = &MessageUpdated{}
	case "permission.asked":
		v = &PermissionAsked{}
	case "permission.replied":
		v = &PermissionReplied{}
	case "session.status":
		v = &SessionStatus{}
	case "session.idle":
		v = &SessionIdle{}
	case "session.error":
		v = &SessionError{}
	default:
		return nil
	}
	if json.Unmarshal(e.Props, v) != nil {
		return nil
	}
	switch t := v.(type) {
	case *PartUpdated:
		return *t
	case *PartDelta:
		return *t
	case *MessageUpdated:
		return *t
	case *PermissionAsked:
		return *t
	case *PermissionReplied:
		return *t
	case *SessionStatus:
		return *t
	case *SessionIdle:
		return *t
	case *SessionError:
		return *t
	}
	return nil
}

// Stream reads /event until ctx ends, reconnecting with backoff. It blocks on the
// network, so it uses no CPU while idle. emit is called from this goroutine.
func (c *Client) Stream(ctx context.Context, emit func(any)) {
	hc := &http.Client{} // no overall timeout: the stream is long-lived
	backoff := time.Second
	for ctx.Err() == nil {
		err := c.streamOnce(ctx, hc, emit)
		if ctx.Err() != nil {
			return
		}
		emit(StreamState{Up: false, Err: err})
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 10*time.Second {
			backoff *= 2
		}
	}
}

func (c *Client) streamOnce(ctx context.Context, hc *http.Client, emit func(any)) error {
	u := c.Base + "/event"
	if c.Dir != "" {
		u += "?" + url.Values{"directory": {c.Dir}}.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return &httpErr{resp.StatusCode}
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64<<10), 8<<20)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var e Event
		if json.Unmarshal([]byte(line[6:]), &e) != nil {
			continue
		}
		if e.Type == "server.connected" {
			emit(StreamState{Up: true})
			continue
		}
		if v := Decode(e); v != nil {
			emit(v)
		}
	}
	return sc.Err()
}

type httpErr struct{ code int }

func (e *httpErr) Error() string { return "event stream returned HTTP " + http.StatusText(e.code) }
