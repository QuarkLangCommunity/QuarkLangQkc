package lang

import (
	"strings"
	"testing"
)

// TestLexRawStringLineNumbers 回归：多行原始字符串之后的行号不再漂移。
// （旧实现 lexRawString 内自增 line 后又走 advance()，每个换行多算一行。）
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
	if bLine != 4 { // 原始字符串占第 2–3 行，int b 在第 4 行
		t.Errorf("多行原始字符串之后的 token 应在第 4 行，got %d", bLine)
	}

	// 两个换行的原始字符串：漂移 2 行的旧行为同样必须消失
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

	// Lex 不受影响：token 流与开关无关
	toks2, err := Lex(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != len(toks2) {
		t.Errorf("Lex 与 LexWithComments 的 token 数应一致: %d vs %d", len(toks2), len(toks))
	}
}

// TestCommentsDoNotShiftLines 回归：块注释跨行不影响其后 token 行号。
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
