//go:build !windows

package xray

import "os/exec"

func configureHiddenCmd(cmd *exec.Cmd) {}
