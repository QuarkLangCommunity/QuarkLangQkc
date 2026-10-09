package main

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Bounds of one bounded phase: how much wall clock and memory a child may take, how often the
// supervisor samples it, and how much of its output is kept.
const (
	outputCap      = 64 << 10
	pollInterval   = 20 * time.Millisecond
	killGrace      = 15 * time.Second
	waitDelay      = 3 * time.Second
	kibPerMiB      = 1024
	positionSample = 200 // characters of a diagnostic quoted into the report
)

// probeResult is the bounded outcome of one child process: how it ended, what it cost and what it said.
type probeResult struct {
	Argv      []string `json:"argv"`
	Exit      int      `json:"exit"`
	WallMS    int64    `json:"wall_ms"`
	PeakRSSKB int64    `json:"peak_rss_kb"` // -1 when the platform cannot report it
	TimedOut  bool     `json:"timed_out"`
	MemCapped bool     `json:"mem_capped"`
	Panicked  bool     `json:"panicked"`
	Stdout    string   `json:"stdout"`
	Stderr    string   `json:"stderr"`
	Truncated bool     `json:"truncated"`
}

// merged is the stream pair the project's own parity script compares: stdout and stderr in order.
func (r probeResult) merged() string { return r.Stdout + r.Stderr }

// seconds renders the wall time the way the report table wants it: seconds with two decimals.
func (r probeResult) seconds() string {
	return strconv.FormatFloat(float64(r.WallMS)/1000, 'f', 2, 64)
}

// peakMiB renders the peak resident set size in mebibytes, or "-" when the platform cannot report it.
func (r probeResult) peakMiB() string {
	if r.PeakRSSKB < 0 {
		return "-"
	}
	return strconv.FormatFloat(float64(r.PeakRSSKB)/1024, 'f', 0, 64)
}

// positionRe matches a source position in either engine's spelling: "at line 12, col 3" from the
// shared front end, "file.kq:12:3" from the compiler's own diagnostics.
var positionRe = regexp.MustCompile(`at line \d+|:\d+:\d+`)

// panicMarkers are the shapes a Go runtime crash leaves behind on stderr.
var panicMarkers = []string{"panic: ", "fatal error: ", "runtime: goroutine stack exceeds", "runtime: out of memory"}

// runBounded starts argv in its own process group, enforces the wall-clock and memory bounds while it
// runs, and returns what it did — including the two kill reasons, which are verdicts in their own right.
// The memory ceiling is in kibibytes, the unit the kernel reports a resident set in.
func runBounded(argv []string, env []string, timeout time.Duration, memCapKB int64) probeResult {
	res := probeResult{Argv: argv, PeakRSSKB: -1}
	var stdout, stderr cappedBuffer
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = env
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.WaitDelay = waitDelay // a grandchild holding the pipe must not hang the supervisor
	setProcessGroup(cmd)

	start := time.Now()
	if err := cmd.Start(); err != nil {
		res.WallMS = time.Since(start).Milliseconds()
		res.Exit = -1
		res.Stderr = "start failed: " + err.Error()
		return res
	}
	pgid := cmd.Process.Pid
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	hardDeadline := start.Add(timeout + killGrace)
	var peakKB int64
	for {
		select {
		case err := <-waited:
			res.WallMS = time.Since(start).Milliseconds()
			res.Exit = exitStatus(err)
			res.PeakRSSKB = maxInt64(peakKB, processPeakRSSKB(cmd.ProcessState))
			res.Stdout, res.Stderr = stdout.String(), stderr.String()
			res.Truncated = stdout.cut() || stderr.cut()
			res.Panicked = looksLikePanic(res.Stderr)
			return res
		case <-ticker.C:
			if rss := processGroupRSSKB(pgid); rss > peakKB {
				peakKB = rss
			}
			if memCapKB > 0 && peakKB > memCapKB && !res.MemCapped {
				res.MemCapped = true
				_ = killGroup(cmd)
			}
			now := time.Now()
			if now.After(start.Add(timeout)) && !res.TimedOut {
				res.TimedOut = true
				_ = killGroup(cmd)
			}
			if now.After(hardDeadline) { // the kill itself did not land: report rather than hang
				res.WallMS = now.Sub(start).Milliseconds()
				res.Exit = -1
				res.Stdout, res.Stderr = stdout.String(), stderr.String()
				res.Truncated = stdout.cut() || stderr.cut()
				return res
			}
		}
	}
}

// looksLikePanic reports whether a stderr stream carries a Go runtime crash rather than a diagnostic.
func looksLikePanic(stderr string) bool {
	for _, marker := range panicMarkers {
		if strings.Contains(stderr, marker) {
			return true
		}
	}
	return false
}

// hasPosition reports whether a stream carries a source position.
func hasPosition(s string) bool { return positionRe.MatchString(s) }

// firstLine returns the first non-empty line, trimmed, for the report's one-line summaries.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			if len(line) > positionSample {
				return line[:positionSample] + "…"
			}
			return line
		}
	}
	return ""
}

// childEnv builds a child environment from the parent's, with the harness overrides replacing any
// inherited value of the same name (duplicate keys would resolve differently per libc).
func childEnv(overrides map[string]string) []string {
	out := make([]string, 0, len(os.Environ())+len(overrides))
	for _, kv := range os.Environ() {
		if key, _, ok := strings.Cut(kv, "="); ok {
			if _, replaced := overrides[key]; replaced {
				continue
			}
		}
		out = append(out, kv)
	}
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out = append(out, key+"="+overrides[key])
	}
	return out
}

// cappedBuffer collects one stream up to outputCap bytes and remembers whether it dropped any.
type cappedBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	truncated bool
}

// Write appends what still fits and records that the stream was cut off.
func (c *cappedBuffer) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	room := outputCap - c.buf.Len()
	if room <= 0 {
		c.truncated = c.truncated || len(p) > 0
		return len(p), nil
	}
	if len(p) > room {
		c.buf.Write(p[:room])
		c.truncated = true
		return len(p), nil
	}
	c.buf.Write(p)
	return len(p), nil
}

// String returns the bytes collected so far.
func (c *cappedBuffer) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

// cut reports whether the stream was truncated.
func (c *cappedBuffer) cut() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.truncated
}

// maxInt64 keeps the larger of two measurements, ignoring the -1 "unknown" sentinel.
func maxInt64(a, b int64) int64 {
	if b < 0 {
		return a
	}
	if a < 0 || b > a {
		return b
	}
	return a
}
