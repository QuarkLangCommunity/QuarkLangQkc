package main

// 编辑器产物校验：VS Code 扩展（package.json / tmLanguage / language-configuration /
// 片段 / JS 语法）与 tree-sitter 语法（tree-sitter.json / queries / 测试语料）。
// 这些产物不是 Go 代码，但属于工具链交付物，放进 go test 才能被 CI 看住。

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func editorsDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "editors")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("缺少 editors 目录: %v", err)
	}
	return dir
}

func readJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s: %v", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("解析 %s: %v", path, err)
	}
}

func TestVSCodePackageManifest(t *testing.T) {
	root := filepath.Join(editorsDir(t), "vscode")
	var pkg struct {
		Name        string   `json:"name"`
		Main        string   `json:"main"`
		Activation  []string `json:"activationEvents"`
		Contributes struct {
			Languages []struct {
				ID            string   `json:"id"`
				Extensions    []string `json:"extensions"`
				Configuration string   `json:"configuration"`
			} `json:"languages"`
			Grammars []struct {
				Language  string `json:"language"`
				ScopeName string `json:"scopeName"`
				Path      string `json:"path"`
			} `json:"grammars"`
			Snippets []struct {
				Language string `json:"language"`
				Path     string `json:"path"`
			} `json:"snippets"`
		} `json:"contributes"`
	}
	readJSON(t, filepath.Join(root, "package.json"), &pkg)

	if pkg.Name != "quarklang" {
		t.Errorf("扩展名应为 quarklang，got %q", pkg.Name)
	}
	if pkg.Main == "" {
		t.Error("缺少 main 入口")
	} else if _, err := os.Stat(filepath.Join(root, pkg.Main)); err != nil {
		t.Errorf("main 入口不存在: %v", err)
	}
	if len(pkg.Contributes.Languages) != 1 || pkg.Contributes.Languages[0].ID != "quarklang" {
		t.Fatalf("语言声明不对: %+v", pkg.Contributes.Languages)
	}
	lang := pkg.Contributes.Languages[0]
	for _, ext := range []string{".qk", ".kq"} {
		found := false
		for _, e := range lang.Extensions {
			if e == ext {
				found = true
			}
		}
		if !found {
			t.Errorf("语言扩展名缺少 %s", ext)
		}
	}
	if _, err := os.Stat(filepath.Join(root, lang.Configuration)); err != nil {
		t.Errorf("language-configuration 不存在: %v", err)
	}
	if len(pkg.Contributes.Grammars) != 1 || pkg.Contributes.Grammars[0].ScopeName != "source.quarklang" {
		t.Fatalf("语法声明不对: %+v", pkg.Contributes.Grammars)
	}
	if _, err := os.Stat(filepath.Join(root, pkg.Contributes.Grammars[0].Path)); err != nil {
		t.Errorf("tmLanguage 不存在: %v", err)
	}
	if len(pkg.Contributes.Snippets) != 1 {
		t.Fatalf("片段声明不对: %+v", pkg.Contributes.Snippets)
	}
	if _, err := os.Stat(filepath.Join(root, pkg.Contributes.Snippets[0].Path)); err != nil {
		t.Errorf("片段文件不存在: %v", err)
	}
}

// TestTextMateGrammarRegexes 逐个编译 tmLanguage 里的正则：
// VS Code 用 Oniguruma，这里用 Go RE2 兜底校验（已避免 lookbehind/lookahead/反向引用）。
func TestTextMateGrammarRegexes(t *testing.T) {
	path := filepath.Join(editorsDir(t), "vscode", "syntaxes", "quarklang.tmLanguage.json")
	var g map[string]interface{}
	readJSON(t, path, &g)
	if g["scopeName"] != "source.quarklang" {
		t.Errorf("scopeName 应为 source.quarklang，got %v", g["scopeName"])
	}
	if langs, _ := g["fileTypes"].([]interface{}); len(langs) == 0 {
		t.Error("缺少 fileTypes")
	}
	n := 0
	var walk func(v interface{}, where string)
	walk = func(v interface{}, where string) {
		switch x := v.(type) {
		case map[string]interface{}:
			for _, key := range []string{"match", "begin", "end"} {
				if s, ok := x[key].(string); ok {
					n++
					if _, err := regexp.Compile(s); err != nil {
						t.Errorf("%s 的 %s 正则不合法（%q）: %v", where, key, s, err)
					}
				}
			}
			for k, sub := range x {
				walk(sub, where+"."+k)
			}
		case []interface{}:
			for i, sub := range x {
				walk(sub, where+"["+itoa(i)+"]")
			}
		}
	}
	walk(g, "$")
	if n < 10 {
		t.Errorf("正则过少（%d），疑似语法文件不完整", n)
	}
}

func TestLanguageConfiguration(t *testing.T) {
	var cfg struct {
		Comments struct {
			LineComment  string   `json:"lineComment"`
			BlockComment []string `json:"blockComment"`
		} `json:"comments"`
		Brackets [][2]string `json:"brackets"`
	}
	readJSON(t, filepath.Join(editorsDir(t), "vscode", "language-configuration.json"), &cfg)
	if cfg.Comments.LineComment != "//" {
		t.Errorf("行注释应为 //，got %q", cfg.Comments.LineComment)
	}
	if len(cfg.Comments.BlockComment) != 2 || cfg.Comments.BlockComment[0] != "/*" {
		t.Errorf("块注释不对: %v", cfg.Comments.BlockComment)
	}
	if len(cfg.Brackets) < 3 {
		t.Errorf("括号对过少: %v", cfg.Brackets)
	}
}

func TestVSCodeSnippets(t *testing.T) {
	var snips map[string]struct {
		Prefix string   `json:"prefix"`
		Body   []string `json:"body"`
	}
	readJSON(t, filepath.Join(editorsDir(t), "vscode", "snippets", "quarklang.json"), &snips)
	for _, want := range []string{"fnmain", "fn", "struct", "interface", "impl", "space", "library", "for", "try", "macro"} {
		found := false
		for _, s := range snips {
			if s.Prefix == want {
				found = true
				if len(s.Body) == 0 {
					t.Errorf("片段 %s 的 body 为空", want)
				}
			}
		}
		if !found {
			t.Errorf("缺少片段 %s", want)
		}
	}
}

func TestVSCodeJSSyntax(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("未安装 node，跳过 JS 语法检查")
	}
	root := filepath.Join(editorsDir(t), "vscode")
	for _, f := range []string{"src/protocol.js", "src/extension.js", "test/protocol.test.js"} {
		cmd := exec.Command(node, "--check", filepath.Join(root, f))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s 语法检查失败: %v\n%s", f, err, out)
		}
	}
}

func TestTreeSitterArtifacts(t *testing.T) {
	root := filepath.Join(editorsDir(t), "tree-sitter-quarklang")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("缺少 tree-sitter 语法目录: %v", err)
	}
	for _, f := range []string{"grammar.js", "tree-sitter.json", "queries/highlights.scm", "queries/locals.scm"} {
		info, err := os.Stat(filepath.Join(root, f))
		if err != nil {
			t.Errorf("缺少 %s: %v", f, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%s 为空", f)
		}
	}
	var cfg struct {
		Grammars []struct {
			Name      string   `json:"name"`
			Scope     string   `json:"scope"`
			FileTypes []string `json:"file-types"`
		} `json:"grammars"`
	}
	readJSON(t, filepath.Join(root, "tree-sitter.json"), &cfg)
	if len(cfg.Grammars) != 1 {
		t.Fatalf("tree-sitter.json 语法条目不对: %+v", cfg.Grammars)
	}
	g := cfg.Grammars[0]
	if g.Name != "quarklang" || g.Scope != "source.quarklang" {
		t.Errorf("tree-sitter 名称/scope 不对: %+v", g)
	}
	for _, ext := range []string{"qk", "kq"} {
		found := false
		for _, e := range g.FileTypes {
			if e == ext {
				found = true
			}
		}
		if !found {
			t.Errorf("tree-sitter file-types 缺少 %s", ext)
		}
	}
	// 至少一个语料测试文件与一条用例
	corpus, err := filepath.Glob(filepath.Join(root, "test", "corpus", "*.txt"))
	if err != nil || len(corpus) == 0 {
		t.Fatalf("缺少 test/corpus/*.txt：%v", err)
	}
	data, err := os.ReadFile(corpus[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "===") || !strings.Contains(string(data), "---") {
		t.Errorf("%s 不是合法的 tree-sitter 语料格式", corpus[0])
	}
}

// itoa 避免再引入 strconv（本文件其他用途不需要）。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
