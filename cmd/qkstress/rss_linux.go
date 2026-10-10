//go:build linux

package main

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// rssProbeSupported reports whether this platform can sample the resident set of a process group.
const rssProbeSupported = true

// processGroupRSSKB sums the resident set size of every process in a group: the compiler drives clang,
// the linker and the linked program as group members, so charging only the direct child would hide the
// memory a case really costs.
func processGroupRSSKB(pgid int) int64 {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return -1
	}
	pageKB := int64(os.Getpagesize()) / 1024
	if pageKB < 1 {
		pageKB = 1
	}
	total := int64(0)
	for _, entry := range entries {
		data, err := os.ReadFile("/proc/" + entry.Name() + "/stat")
		if err != nil {
			continue
		}
		// The comm field may hold spaces and parentheses, so only the tail after the last ')' is stable:
		// it starts at field 3 (state), which puts pgrp at index 2 and rss (field 24) at index 21.
		fields := strings.Fields(string(data)[strings.LastIndexByte(string(data), ')')+1:])
		if len(fields) < 22 {
			continue
		}
		if pgrp, err := strconv.Atoi(fields[2]); err != nil || pgrp != pgid {
			continue
		}
		if rss, err := strconv.ParseInt(fields[21], 10, 64); err == nil && rss > 0 {
			total += rss * pageKB
		}
	}
	return total
}

// processPeakRSSKB returns the peak resident set size the kernel accounted to the direct child (Linux
// reports it in kibibytes).
func processPeakRSSKB(state *os.ProcessState) int64 {
	if state == nil {
		return -1
	}
	if usage, ok := state.SysUsage().(*syscall.Rusage); ok && usage.Maxrss > 0 {
		return usage.Maxrss
	}
	return -1
}
