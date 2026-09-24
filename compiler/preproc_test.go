package main

import "testing"

func TestPreprocPlatformSwitch(t *testing.T) {
	src := `#if os("windows")
io.println(3);
#elif os("darwin")
io.println(2);
#else
io.println(1);
#endif`
	linux := newPreprocCtx()
	linux.os = "linux"
	out, err := linux.Process(src, "t.qk")
	if err != nil {
		t.Fatal(err)
	}
	if out != "io.println(1);" {
		t.Fatalf("linux 分支错误：%q", out)
	}
	win := newPreprocCtx()
	win.os = "windows"
	out, err = win.Process(src, "t.qk")
	if err != nil {
		t.Fatal(err)
	}
	if out != "io.println(3);" {
		t.Fatalf("windows 分支错误：%q", out)
	}
}

func TestPreprocDefinesAndArch(t *testing.T) {
	src := `#define MARKET
#if arch("arm64") && defined(MARKET)
arm;
#endif
#if !defined(NOPE) || os("plan9")
yes;
#endif`
	ctx := newPreprocCtx()
	ctx.arch = "arm64"
	out, err := ctx.Process(src, "t.qk")
	if err != nil {
		t.Fatal(err)
	}
	if out != "arm;\nyes;" {
		t.Fatalf("定义/架构条件错误：%q", out)
	}
}

func TestPreprocErrorDirective(t *testing.T) {
	ctx := newPreprocCtx()
	ctx.os = "windows"
	if _, err := ctx.Process("#if os(\"windows\")\n#error nope\n#endif", "t.qk"); err == nil {
		t.Fatal("#error 应触发失败")
	}
	ctx2 := newPreprocCtx()
	ctx2.os = "linux"
	if _, err := ctx2.Process("#if os(\"windows\")\n#error nope\n#endif", "t.qk"); err != nil {
		t.Fatalf("非目标平台不应报错：%v", err)
	}
}

func TestPreprocUnclosed(t *testing.T) {
	ctx := newPreprocCtx()
	if _, err := ctx.Process("#if os(\"linux\")\nx;\n", "t.qk"); err == nil {
		t.Fatal("未闭合的 #if 应报错")
	}
}
