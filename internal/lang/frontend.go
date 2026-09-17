package lang

// ============ 单文件前端入口（工具共用） ============
//
// 语言本身的正式入口是 Compile / CompileWithImports（含类型检查与 import 合并）。
// 工具（qkcheck / qkdoc / qkrepl / qklsp）常常需要**单文件、原坐标**的 AST：
// 这里只做 词法 → 宏展开 → 解析，不做类型检查、不解析 import。

// ParseSource 只做前端：词法 → 宏展开 → 解析（不解析 import、不做类型检查）。
// 行号即文件行号（合并 import 后的坐标见 SrcMap）。
func ParseSource(src string) (*Program, error) {
	prog, _, err := ParseSourceWithComments(src)
	return prog, err
}

// ParseSourceWithComments 同 ParseSource，另返回源码中的注释（按出现顺序）。
func ParseSourceWithComments(src string) (*Program, []Comment, error) {
	toks, comments, err := LexWithComments(src)
	if err != nil {
		return nil, nil, err
	}
	macros, rest, err := SplitMacroDefs(toks)
	if err != nil {
		return nil, nil, err
	}
	if len(macros) > 0 {
		rest, err = ExpandMacros(rest, macros, "explain")
		if err != nil {
			return nil, nil, err
		}
	}
	prog, err := Parse(rest)
	if err != nil {
		return nil, nil, err
	}
	prog.Src = src
	return prog, comments, nil
}
