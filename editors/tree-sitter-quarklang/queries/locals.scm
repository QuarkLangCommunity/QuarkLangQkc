; 作用域与定义/引用（Neovim 的 locals 支持，用于高亮变量与重命名）
(function_declaration) @local.scope
(block) @local.scope

(function_declaration name: (identifier) @local.definition)
(parameter name: (identifier) @local.definition)
(declaration_statement name: (identifier) @local.definition)
(for_in_statement name: (identifier) @local.definition)

(identifier) @local.reference
