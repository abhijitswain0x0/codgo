//go:build windows

package main

import "os/exec"

func setProcAttr(c *exec.Cmd) {}

func killTree(c *exec.Cmd) { _ = c.Process.Kill() }
