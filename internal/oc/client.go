// Package oc is a thin HTTP + SSE client for `opencode serve`.
package oc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	Base string // e.g. http://127.0.0.1:4096
	Dir  string // project directory sent as ?directory=
	hc   *http.Client
}

func New(base, dir string) *Client {
	return &Client{Base: strings.TrimRight(base, "/"), Dir: dir, hc: &http.Client{Timeout: 30 * time.Second}}
}

type Session struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Directory string `json:"directory"`
	Time      struct {
		Created int64 `json:"created"`
		Updated int64 `json:"updated"`
	} `json:"time"`
}

type Model struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Cost struct {
		Input float64 `json:"input"`
	} `json:"cost"`
}

type Provider struct {
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Models map[string]Model `json:"models"`
}

type Providers struct {
	Providers []Provider        `json:"providers"`
	Default   map[string]string `json:"default"`
}

type ModelRef struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

func (c *Client) do(ctx context.Context, method, path string, q url.Values, in, out any) error {
	u := c.Base + path
	if q == nil {
		q = url.Values{}
	}
	if c.Dir != "" && q.Get("directory") == "" {
		q.Set("directory", c.Dir)
	}
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode/100 != 2 {
		msg := strings.TrimSpace(string(data))
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, msg)
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

func (c *Client) Health(ctx context.Context) error {
	return c.do(ctx, "GET", "/global/health", nil, nil, nil)
}

func (c *Client) CreateSession(ctx context.Context) (Session, error) {
	var s Session
	err := c.do(ctx, "POST", "/session", nil, map[string]any{}, &s)
	return s, err
}

func (c *Client) ListSessions(ctx context.Context) ([]Session, error) {
	var s []Session
	err := c.do(ctx, "GET", "/session", url.Values{"roots": {"true"}}, nil, &s)
	return s, err
}

func (c *Client) Providers(ctx context.Context) (Providers, error) {
	var p Providers
	err := c.do(ctx, "GET", "/config/providers", nil, nil, &p)
	return p, err
}

// Prompt queues a user message. Output arrives on the event stream.
func (c *Client) Prompt(ctx context.Context, sid string, m *ModelRef, text string) error {
	body := map[string]any{"parts": []map[string]string{{"type": "text", "text": text}}}
	if m != nil && m.ModelID != "" {
		body["model"] = m
	}
	return c.do(ctx, "POST", "/session/"+sid+"/prompt_async", nil, body, nil)
}

func (c *Client) Abort(ctx context.Context, sid string) error {
	return c.do(ctx, "POST", "/session/"+sid+"/abort", nil, map[string]any{}, nil)
}

// Reply answers a permission request: once, always or reject.
func (c *Client) Reply(ctx context.Context, id, reply, message string) error {
	body := map[string]string{"reply": reply}
	if message != "" {
		body["message"] = message
	}
	return c.do(ctx, "POST", "/permission/"+id+"/reply", nil, body, nil)
}

// Pending lists permission requests that are still waiting.
func (c *Client) Pending(ctx context.Context) ([]PermissionAsked, error) {
	var p []PermissionAsked
	err := c.do(ctx, "GET", "/permission", nil, nil, &p)
	return p, err
}

type Message struct {
	Info  MessageInfo `json:"info"`
	Parts []Part      `json:"parts"`
}

func (c *Client) Messages(ctx context.Context, sid string) ([]Message, error) {
	var m []Message
	err := c.do(ctx, "GET", "/session/"+sid+"/message", nil, nil, &m)
	return m, err
}
