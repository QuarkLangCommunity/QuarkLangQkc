package i18n

import (
	"regexp"
	"testing"
)

// TestVerbSequenceMatches is the ordered counterpart of TestVerbSetsMatch.
//
// A translation must keep the placeholders in the same order as the source: call sites pass their
// arguments positionally, so a translation with the right verbs in the wrong order still passes the
// set check and then renders nonsense at run time. That is exactly what happened when a batch of
// compiler messages was migrated to English source and entries moved %s around.
//
// It lands as a ratchet: the existing mismatches (found by this test the first time it ran) may only
// be fixed, never added to. Each message that appears twice in the catalog — the English source key
// and the legacy Chinese key — has to be handled as a pair, which is why the repair is done in small
// batches rather than one sweep.
func TestVerbSequenceMatches(t *testing.T) {
	const allowed = 12 // measured when the test was added; fix in batches, never grow
	re := regexp.MustCompile("%[a-zA-Z]")
	bad, shown := 0, 0
	for _, domain := range stdCatalog.Domains() {
		for template, translated := range stdCatalog.Domain(domain) {
			a, b := re.FindAllString(template, -1), re.FindAllString(translated, -1)
			if len(a) != len(b) {
				continue // the set check reports the count mismatch
			}
			for i := range a {
				if a[i] != b[i] {
					bad++
					if shown < 6 {
						shown++
						t.Logf("%s: order differs at #%d\n  source: %q -> %v\n  target: %q -> %v",
							domain, i+1, truncate(template), a, truncate(translated), b)
					}
					break
				}
			}
		}
	}
	if bad > allowed {
		t.Fatalf("%d catalog entries reorder their placeholders (limit %d); the translation must follow the source order", bad, allowed)
	}
	t.Logf("catalog entries with reordered placeholders: %d (limit %d)", bad, allowed)
}
