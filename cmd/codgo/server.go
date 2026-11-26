package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/abhijitswain0x0/codgo/internal/oc"
)

type server struct{ cmd *exec.Cmd }

// Stop ends the server only if codgo started it.
func (s *server) Stop() {
	if s == nil || s.cmd == nil || s.cmd.Process == nil {
		return
	}
	killTree(s.cmd)
	s.cmd.Wait()
	s.cmd = nil
}

func healthy(base string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	return oc.New(base, "").Health(ctx) == nil
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// ensureServer uses url if given, otherwise the default local server if it is up,
// otherwise starts `opencode serve` on a free port.
func ensureServer(url, cwd string) (*server, string, error) {
	if url != "" {
		if !healthy(url) {
			return nil, "", fmt.Errorf("no opencode server answering at %s", url)
		}
		return &server{}, url, nil
	}
	def := "http://127.0.0.1:4096"
	if healthy(def) {
		return &server{}, def, nil
	}
	bin, err := exec.LookPath("opencode")
	if err != nil {
		return nil, "", fmt.Errorf("opencode not found on PATH (install it, or pass --url)")
	}
	port, err := freePort()
	if err != nil {
		return nil, "", err
	}
	cmd := exec.Command(bin, "serve", "--hostname", "127.0.0.1", "--port", fmt.Sprint(port))
	cmd.Dir = cwd
	cmd.Env = os.Environ()
	if os.Getenv("OPENCODE_CONFIG_CONTENT") == "" {
		cmd.Env = append(cmd.Env, `OPENCODE_CONFIG_CONTENT={"permission":{"bash":"ask","edit":"ask"}}`)
	}
	fmt.Fprintln(os.Stderr, "codgo: starting opencode serve...")
	setProcAttr(cmd)
	if err := cmd.Start(); err != nil {
		return nil, "", err
	}
	s := &server{cmd: cmd}
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) { // startup only, not a polling loop at runtime
		if healthy(base) {
			return s, base, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	s.Stop()
	return nil, "", fmt.Errorf("opencode serve did not become ready")
}
