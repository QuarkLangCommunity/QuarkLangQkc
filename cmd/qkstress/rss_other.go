//go:build !linux

package main

import "os"

// rssProbeSupported reports whether this platform can sample the resident set of a process group.
const rssProbeSupported = false

// processGroupRSSKB is the non-Linux spelling: without /proc the harness cannot sample a group, and -1
// marks the measurement as unavailable rather than zero.
func processGroupRSSKB(pgid int) int64 { return -1 }

// processPeakRSSKB is the non-Linux spelling: the portable ProcessState carries no peak RSS figure.
func processPeakRSSKB(state *os.ProcessState) int64 { return -1 }
