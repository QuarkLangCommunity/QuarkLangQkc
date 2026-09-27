package i18n

import "sync"

// Missing translations are counted instead of being swallowed: a call site that forgets to
// register its template still renders (falling back to the Chinese source text), but tests can
// assert that the count stays zero, which turns "silent fallback" into a failure.
var (
	missMu    sync.Mutex
	missCount map[string]int
)

func countMiss(template string) {
	missMu.Lock()
	if missCount == nil {
		missCount = make(map[string]int)
	}
	missCount[template]++
	missMu.Unlock()
}

// Misses returns how often each unregistered template was requested, and clears the counters.
func Misses() map[string]int {
	missMu.Lock()
	defer missMu.Unlock()
	out := missCount
	missCount = nil
	return out
}

// MissTotal returns the total number of unregistered lookups since the last Misses call.
func MissTotal() int {
	missMu.Lock()
	defer missMu.Unlock()
	n := 0
	for _, c := range missCount {
		n += c
	}
	return n
}
