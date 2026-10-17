package statusgen

import (
	"path/filepath"
	"testing"
)

// TestGeneratedStatusIsCurrent is the standing drift gate: the committed report must be exactly what the
// claims and template render, so a test, a corpus case or a claim edited without regenerating fails the
// ordinary go test run.
func TestGeneratedStatusIsCurrent(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(Options{Root: root, Check: true}); err != nil {
		t.Fatal(err)
	}
}
