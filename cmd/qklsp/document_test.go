package main

// 跨系统：LSP 文件 URI 与本地路径互转（Windows 盘符 / POSIX 路径 / 百分号编码）。
import (
	"strings"
	"testing"
)

func TestPathToURISlash(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/home/jack/a.qk", "/home/jack/a.qk"},
		{"C:/Users/jack/a.qk", "/C:/Users/jack/a.qk"}, // Windows：补引导斜杠 → file:///C:/...
		{"D:/x.qk", "/D:/x.qk"},
	}
	for _, c := range cases {
		if got := uriSlashPath(c.in); got != c.want {
			t.Errorf("uriSlashPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPathFromURISlash(t *testing.T) {
	// 非 Windows 平台：引导斜杠保留（/C:/x 在 POSIX 上就是普通路径）
	if got := pathFromURISlash("/home/jack/a.qk"); got != "/home/jack/a.qk" {
		t.Errorf("POSIX 路径不应被改写，got %q", got)
	}
	// Windows 分支的行为单独核对（在非 Windows 平台上不触发，逻辑由 uriSlashPath 覆盖）
	if got := uriSlashPath("C:/x"); got != "/C:/x" {
		t.Errorf("Windows 盘符应补斜杠，got %q", got)
	}
}

func TestURIRoundTrip(t *testing.T) {
	// 相对路径 → URI → 路径：至少保证 file:// 前缀与三段斜杠（跨平台一致）
	uri := pathToURI("x.qk")
	if len(uri) < len("file:///") || uri[:8] != "file:///" {
		t.Errorf("URI 应为 file:///<abs> 形式，got %q", uri)
	}
	back := uriToPath(uri)
	if !strings.HasSuffix(back, "x.qk") {
		t.Errorf("往返后应仍指向 x.qk，got %q", back)
	}
}
