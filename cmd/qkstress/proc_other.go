//go:build !unix

package main

import (
	"errors"
	"os/exec"
)

// setProcessGroup is the non-Unix spelling: process groups do not exist, so the child runs alone.
func setProcessGroup(cmd *exec.Cmd) {}

// killGroup kills the child, which is the whole tree the harness can reach on this platform.
func killGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}

// exitStatus turns a Wait error into an exit status.
func exitStatus(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
