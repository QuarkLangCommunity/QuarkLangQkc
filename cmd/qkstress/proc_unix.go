//go:build unix

package main

import (
	"errors"
	"os/exec"
	"syscall"
)

// setProcessGroup puts the child in its own process group, so one bounded kill reaches everything the
// child started — for the compiler that is clang, the linker and the linked program.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killGroup kills the child's whole process group and falls back to the child alone.
func killGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		return cmd.Process.Kill()
	}
	return nil
}

// exitStatus turns a Wait error into an exit status, keeping a death by signal visible as 128+signal.
func exitStatus(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			return 128 + int(status.Signal())
		}
		return exitErr.ExitCode()
	}
	return -1
}
