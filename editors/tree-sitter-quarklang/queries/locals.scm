; Scopes and definitions/references (Neovim locals support: variable highlighting and renaming)
(function_declaration) @local.scope
(block) @local.scope

(function_declaration name: (identifier) @local.definition)
(parameter name: (identifier) @local.definition)
(declaration_statement name: (identifier) @local.definition)
(for_in_statement name: (identifier) @local.definition)

(identifier) @local.reference
