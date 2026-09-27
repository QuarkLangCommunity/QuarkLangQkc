package lang

import (
	"strings"
	"testing"
)

// TestLexRawStringLineNumbers regression: line numbers no longer drift after a multi-line raw string.
// (The old implementation incremented line inside lexRawString and then also ran advance(), so every newline added one line too many.)
func TestLexRawStringLineNumbers(t *testing.T) {
	src := "String a = \"x\";\nString r = `raw\nline`;\nint b = 1;\n"
	toks, err := Lex(src)
	if err != nil {
		t.Fatal(err)
	}
	var bLine int
	for _, tk := range toks {
		if tk.Kind == TIdent && tk.Text == "b" {
			bLine = tk.Line
		}
	}
	if bLine != 4 { // the raw string occupies lines 2-3, int b is on line 4
		t.Errorf("多行原始字符串之后的 token 应在第 4 行，got %d", bLine)
	}

	// raw string with two newlines: the old 2-line drift must disappear as well
	src2 := "String r = `a\nb\nc`;\nint b = 1;\n"
	toks2, err := Lex(src2)
	if err != nil {
		t.Fatal(err)
	}
	for _, tk := range toks2 {
		if tk.Kind == TIdent && tk.Text == "b" {
			if tk.Line != 4 {
				t.Errorf("应第 4 行，got %d", tk.Line)
			}
		}
	}
}

func TestLexWithComments(t *testing.T) {
	src := "/* 文件头\n   第二行 */\n// 行注释\nfn main(IOStream io) {\n    int x = 1; // 尾注释\n}\n"
	toks, comments, err := LexWithComments(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) == 0 {
		t.Fatal("token 为空")
	}
	if len(comments) != 3 {
		t.Fatalf("应收集 3 条注释，got %d: %+v", len(comments), comments)
	}
	if !comments[0].Block || comments[0].Pos.Line != 1 || comments[0].EndLine != 2 {
		t.Errorf("块注释位置/结束行不对: %+v", comments[0])
	}
	if !strings.Contains(comments[0].Text, "文件头") || !strings.Contains(comments[0].Text, "*/") {
		t.Errorf("块注释原文应含内容与定界符: %q", comments[0].Text)
	}
	if comments[1].Block || comments[1].Pos.Line != 3 {
		t.Errorf("行注释识别不对: %+v", comments[1])
	}
	if comments[2].Pos.Line != 5 || !strings.HasSuffix(comments[2].Text, "// 尾注释") {
		t.Errorf("行尾注释识别不对: %+v", comments[2])
	}

	// Lex is unaffected: the token stream does not depend on the switch
	toks2, err := Lex(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != len(toks2) {
		t.Errorf("Lex 与 LexWithComments 的 token 数应一致: %d vs %d", len(toks2), len(toks))
	}
}

// TestCommentsDoNotShiftLines regression: a block comment spanning several lines does not affect the line numbers of the tokens after it.
func TestCommentsDoNotShiftLines(t *testing.T) {
	src := "/* a\nb\nc */\nint x = 1;\n"
	toks, err := Lex(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, tk := range toks {
		if tk.Kind == TIdent && tk.Text == "x" {
			if tk.Line != 4 {
				t.Errorf("注释之后的 token 应第 4 行，got %d", tk.Line)
			}
		}
	}
}
