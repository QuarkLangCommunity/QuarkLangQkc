package main

// Cross-system: LSP file URI <-> local path conversion (Windows drive / POSIX path / percent-encoding).
import (
	"strings"
	"testing"
)

func TestPathToURISlash(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/home/user/a.qk", "/home/user/a.qk"},
		{"C:/Users/user/a.qk", "/C:/Users/user/a.qk"}, // Windows: prepend the leading slash -> file:///C:/...
		{"D:/x.qk", "/D:/x.qk"},
	}
	for _, c := range cases {
		if got := uriSlashPath(c.in); got != c.want {
			t.Errorf("uriSlashPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPathFromURISlash(t *testing.T) {
	// Non-Windows platforms: the leading slash is kept (/C:/x is just an ordinary path on POSIX)
	if got := pathFromURISlash("/home/user/a.qk"); got != "/home/user/a.qk" {
		t.Errorf("a POSIX path should not be rewritten, got %q", got)
	}
	// The Windows-only branch is reviewed separately (it does not trigger on non-Windows platforms; the logic is covered by uriSlashPath)
	if got := uriSlashPath("C:/x"); got != "/C:/x" {
		t.Errorf("a Windows drive letter should get a trailing slash, got %q", got)
	}
}

func TestURIRoundTrip(t *testing.T) {
	// Relative path -> URI -> path: at least guarantee the file:// prefix and the three slashes (consistent across platforms)
	uri := pathToURI("x.qk")
	if len(uri) < len("file:///") || uri[:8] != "file:///" {
		t.Errorf("the URI should have the form file:///<abs>, got %q", uri)
	}
	back := uriToPath(uri)
	if !strings.HasSuffix(back, "x.qk") {
		t.Errorf("after a round trip it should still point at x.qk, got %q", back)
	}
}
