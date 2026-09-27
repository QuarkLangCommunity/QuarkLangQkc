; QuarkLang syntax highlighting queries (tree-sitter)
; Editors (Neovim / Helix / Emacs, ...) read highlight names from this file.

; ---- Comments and literals ----
(line_comment) @comment
(block_comment) @comment

(string) @string
(raw_string) @string.special
(number) @number
(boolean) @constant.builtin
(null) @constant.builtin

; ---- Fallback: plain identifiers as variables (more specific rules below override this) ----
(identifier) @variable

; ---- Declarations ----
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

; ---- Statement keywords ----
["if" "else" "while" "for" "break" "return" "log" "delete" "try" "catch" "new"] @keyword.control
"@" @operator

; ---- Types ----
(builtin_type) @type.builtin

; ---- Names (declaration sites) ----
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

; ---- Calls and members ----
; Note: the function field is wrapped as expression → primary (see the suffix-chain shape in grammar.js)
(call_expression function: (expression (primary (identifier) @function.call)))
(call_expression function: (expression (primary (member_expression name: (identifier) @function.method))))
(scope_call scope: (identifier) @namespace)
(scope_call name: (identifier) @function.call)
(member_expression name: (identifier) @property)
(struct_literal (struct_field name: (identifier) @property))


; ---- Macros ----
(macro_definition name: (identifier) @function.macro)
(macro_call name: (identifier) @function.macro)
(directive_statement "#" @keyword.directive)
(directive_statement (identifier) @keyword.directive)

; ---- Signature calls @mb() ----
(sign_call name: (identifier) @variable)

; ---- Operators and punctuation ----
(binary_expression operator: _ @operator)
(unary_expression operator: _ @operator)
["=" "::" "." "&"] @operator
["(" ")" "[" "]" "{" "}"] @punctuation.bracket
["," ";" ":"] @punctuation.delimiter
