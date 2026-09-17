; QuarkLang 语法高亮查询（tree-sitter）
; 编辑器（Neovim / Helix / Emacs 等）通过本文件取高亮名。

; ---- 注释与字面量 ----
(line_comment) @comment
(block_comment) @comment

(string) @string
(raw_string) @string.special
(number) @number
(boolean) @constant.builtin
(null) @constant.builtin

; ---- 兜底：普通标识符当变量（后续更具体的规则会覆盖它） ----
(identifier) @variable

; ---- 声明 ----
"fn" @keyword.function
"type" @keyword.type
"struct" @keyword.type
"interface" @keyword.type
"impl" @keyword
"space" @keyword
"library" @keyword
"program" @keyword.directive
"import" @keyword.import
"pub" @keyword.modifier
"dynamic" @keyword.modifier
"const" @keyword.modifier
"copyd" @keyword.modifier
"expand" @keyword

; ---- 语句关键字 ----
["if" "else" "while" "for" "break" "return" "log" "delete" "try" "catch" "new"] @keyword.control
"@" @operator

; ---- 类型 ----
(builtin_type) @type.builtin

; ---- 名字（声明点） ----
(function_declaration name: (identifier) @function)
(method_signature name: (identifier) @function.method)
(library_method name: (identifier) @function)
(struct_declaration name: (identifier) @type)
(interface_declaration name: (identifier) @type)
(impl_declaration name: (identifier) @type)
(space_declaration name: (identifier) @namespace)
(library_declaration name: (identifier) @namespace)
(type_alias name: (identifier) @type)
(parameter name: (identifier) @variable.parameter)
(declaration_statement name: (identifier) @variable)
(member_declaration name: (identifier) @property)
(struct_field name: (identifier) @property)

; ---- 调用与成员 ----
; 注意：function 字段是 expression → primary 包装（见 grammar.js 的后缀链形态）
(call_expression function: (expression (primary (identifier) @function.call)))
(call_expression function: (expression (primary (member_expression name: (identifier) @function.method))))
(scope_call scope: (identifier) @namespace)
(scope_call name: (identifier) @function.call)
(member_expression name: (identifier) @property)
(struct_literal (struct_field name: (identifier) @property))


; ---- 宏 ----
(macro_definition name: (identifier) @function.macro)
(macro_call name: (identifier) @function.macro)
(directive_statement "#" @keyword.directive)
(directive_statement (identifier) @keyword.directive)

; ---- 签名调用 @mb() ----
(sign_call name: (identifier) @variable)

; ---- 运算符与标点 ----
(binary_expression operator: _ @operator)
(unary_expression operator: _ @operator)
["=" "::" "." "&"] @operator
["(" ")" "[" "]" "{" "}"] @punctuation.bracket
["," ";" ":"] @punctuation.delimiter
