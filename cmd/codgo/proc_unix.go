//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func setProcAttr(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

func killTree(c *exec.Cmd) {
	// negative pid signals the whole process group (the node wrapper and its child)
	_ = syscall.Kill(-c.Process.Pid, syscall.SIGTERM)
}
