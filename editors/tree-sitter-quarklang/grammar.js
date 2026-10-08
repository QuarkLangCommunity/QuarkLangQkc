/**
 * QuarkLang grammar (tree-sitter)
 *
 * Canonical syntax lives in the repo's SYNTAX.md: type first (`int x`), `fn name(int a) Ret`,
 * `type struct { ... } Name;`, `impl { ... } Name;`, `space { ... } name;`,
 * `for (int x : l)`, `catch (void e)`, `macro name($a $b)`, `@signature` calls, and so on.
 */
module.exports = grammar({
  name: 'quarklang',

  extras: $ => [
    /\s|\\\r?\n/,
    $.line_comment,
    $.block_comment,
  ],

  word: $ => $.identifier,

  conflicts: $ => [
    [$.new_expression],
    [$.primary, $.type_base],
  ],

  supertypes: $ => [],

  rules: {
    source_file: $ => repeat($._top_level),

    _top_level: $ => choice(
      $.program_declaration,
      $.import_declaration,
      $.function_declaration,
      $.struct_declaration,
      $.interface_declaration,
      $.impl_declaration,
      $.space_declaration,
      $.library_declaration,
      $.type_alias,
      $.macro_definition,
      $.pub_declaration,
    ),

    // program main; / program library;
    program_declaration: $ => seq('program', $.identifier, ';'),

    // import "path"; / import path;
    import_declaration: $ => seq('import', choice($.string, $.identifier), ';'),

    // pub fn|type struct|type interface ...: makes the next symbol public
    pub_declaration: $ => seq(
      'pub',
      choice($.function_declaration, $.struct_declaration, $.interface_declaration),
    ),

    // fn name<T>(params) Ret { ... }
    function_declaration: $ => seq(
      'fn',
      optional($.type_parameters),
      field('name', $.identifier),
      field('parameters', $.parameter_list),
      optional(field('return_type', $.type)),
      field('body', $.block),
    ),

    parameter_list: $ => seq('(', optional(commaSep1($.parameter)), ')'),
    parameter: $ => seq(optional($.decorator_modifier), field('type', $.type), field('name', $.identifier)),

    decorator_modifier: _ => choice('const', 'copyd'),

    type_parameters: $ => seq('<', commaSep1($.identifier), '>'),

    // type struct { ... } Name;
    struct_declaration: $ => seq(
      'type',
      'struct',
      optional($.type_parameters),
      '{',
      repeat($.member_declaration),
      '}',
      field('name', $.identifier),
      ';',
    ),

    member_declaration: $ => seq(field('type', $.type), field('name', $.identifier), ';'),

    // type interface { fn m(Self self) Ret; expand interface Other; } Name;
    interface_declaration: $ => seq(
      'type',
      'interface',
      optional($.type_parameters),
      '{',
      repeat(choice($.method_signature, $.expand_declaration)),
      '}',
      field('name', $.identifier),
      ';',
    ),

    method_signature: $ => seq(
      optional('dynamic'), // dynamic modifier: interface methods are dispatched dynamically at runtime
      'fn',
      field('name', $.identifier),
      seq('(', optional(commaSep1($.parameter)), ')'),
      optional(field('return_type', $.type)),
      ';',
    ),

    expand_declaration: $ => seq('expand', 'interface', $.identifier, ';'),

    // impl<T> { ... } Type;
    impl_declaration: $ => seq(
      'impl',
      optional($.type_parameters),
      '{',
      repeat($.function_declaration),
      '}',
      field('name', $.identifier),
      ';',
    ),

    // space { ... } name;
    space_declaration: $ => seq(
      'space',
      '{',
      repeat($.function_declaration),
      '}',
      field('name', $.identifier),
      ';',
    ),

    // library name { fn sym(int a) Ret; ... }
    library_declaration: $ => seq(
      'library',
      field('name', $.identifier),
      '{',
      repeat($.library_method),
      '}',
    ),

    library_method: $ => seq(
      'fn',
      field('name', $.identifier),
      seq('(', optional(commaSep1($.parameter)), ')'),
      optional(field('return_type', $.type)),
      ';',
    ),

    // type <type> name;
    type_alias: $ => seq('type', field('type', $.type), field('name', $.identifier), ';'),

    // macro name($a $b) { body }
    macro_definition: $ => seq(
      'macro',
      field('name', $.identifier),
      '(',
      repeat1(seq('$', $.identifier)),
      ')',
      field('body', $.macro_block),
    ),

    // A macro body is token-level content (may contain #when/#return/#error and nested blocks), parsed permissively
    macro_block: $ => seq('{', repeat(choice($._macro_token, $.macro_block)), '}'),
    _macro_token: $ => choice(
      $.identifier, $.number, $.string, $.raw_string, $.line_comment, $.block_comment,
      '(', ')', '[', ']', ',', ';', ':', '.', '=', '+', '-', '*', '/', '%',
      '<', '>', '!', '&', '|', '?', '@', '#', '$', '::',
    ),

    // ---- Statements ----
    block: $ => seq('{', repeat($._statement), '}'),

    _statement: $ => choice(
      $.declaration_statement,
      $.expression_statement,
      $.if_statement,
      $.while_statement,
      $.for_in_statement,
      $.for_c_statement,
      $.return_statement,
      $.log_statement,
      $.delete_statement,
      $.break_statement,
      $.try_statement,
      $.directive_statement,
    ),

    // int x = 1; / Point p; / const int N = 3; / copyd List<int> l = [1];
    declaration_statement: $ => seq(
      optional($.decorator_modifier),
      field('type', $.type),
      field('name', $.identifier),
      optional(seq('=', field('value', $.expression))),
      ';',
    ),

    // Assignment is an expression (C style): x = 1 / p.x = 3 / l[i] = 1 / *p = v
    // The left-hand side uses expression directly (covering index/member/unary) to avoid static
    // resolution issues caused by choice order
    assignment_expression: $ => prec.right(1, seq(
      field('left', $.expression),
      '=',
      field('right', $.expression),
    )),

    expression_statement: $ => seq($.expression, ';'),

    if_statement: $ => prec.right(seq(
      'if', '(', field('condition', $.expression), ')', field('consequence', $.block),
      optional(seq('else', field('alternative', choice($.block, $.if_statement)))),
    )),

    while_statement: $ => seq('while', '(', field('condition', $.expression), ')', field('body', $.block)),

    // for (int x : list) { }
    for_in_statement: $ => seq(
      'for', '(',
      field('type', $.type),
      field('name', $.identifier),
      ':',
      field('iterable', $.expression),
      ')',
      field('body', $.block),
    ),

    // for (int i = 0; i < n; i = i + 1) { }
    for_c_statement: $ => seq(
      'for', '(',
      field('init', choice($.declaration_statement, $.expression_statement, ';')),
      optional(field('condition', $.expression)),
      ';',
      optional(field('update', $.expression)),
      ')',
      field('body', $.block),
    ),

    return_statement: $ => seq('return', optional($.expression), ';'),
    log_statement: $ => seq('log', $.expression, ';'),
    delete_statement: $ => seq('delete', $.expression, ';'),
    break_statement: $ => seq('break', optional(';')),

    try_statement: $ => seq(
      'try', field('body', $.block),
      'catch', '(', field('error_type', $.type), field('error_name', $.identifier), ')',
      field('handler', $.block),
    ),

    directive_statement: $ => seq('#', $.identifier, optional(seq('(', optional(commaSep1($.expression)), ')')), ';'),

    // ---- Expressions ----
    // Uses the "primary + suffix chain" shape: `a.b(c)[d].e` is carried recursively by _primary,
    // avoiding static resolution errors from "reduce to expression first, then decide whether to consume `[`".
    expression: $ => choice(
      $.assignment_expression,
      $.binary_expression,
      $.unary_expression,
      $.primary,
    ),

    primary: $ => choice(
      $.call_expression,
      $.macro_call,
      $.member_expression,
      $.index_expression,
      $.scope_call,
      $.struct_literal,
      $.list_literal,
      $.new_expression,
      $.sign_call,
      $.parenthesized_expression,
      $.identifier,
      $.number,
      $.string,
      $.raw_string,
      $.boolean,
      $.null,
    ),

    parenthesized_expression: $ => seq('(', $.expression, ')'),

    binary_expression: $ => choice(
      ...[
        ['||', 1],
        ['&&', 2],
        ['==', 3], ['!=', 3],
        ['<', 4], ['<=', 4], ['>', 4], ['>=', 4],
        ['<<', 5], ['>>', 5],
        ['+', 6], ['-', 6],
        ['*', 7], ['/', 7], ['%', 7],
      ].map(([op, level]) => prec.left(level, seq(
        field('left', $.expression), field('operator', op), field('right', $.expression),
      ))),
    ),

    unary_expression: $ => prec(8, seq(
      field('operator', choice('!', '-', '*', '&')),
      field('operand', $.expression),
    )),

    call_expression: $ => prec(10, seq(
      field('function', $.expression),
      field('arguments', $.argument_list),
      optional($.sign_call),
    )),

    argument_list: $ => seq('(', optional(commaSep1($.expression)), ')'),

    // Macro calls accept three delimiters: name(args) / name[args] / name{args} (M1)
    // Only the [] and {} forms are handled here (the () form is covered by call_expression).
    // Only the {} form is kept: the [] form conflicts with index access (a macro call name[a, b] parses as index syntax, which is fine for highlighting)
    macro_call: $ => prec(12, seq(
      field('name', $.identifier), '{', optional(commaSep1($.expression)), '}',
    )),

    // f(args) @mb(args)
    sign_call: $ => seq('@', field('name', $.identifier), $.argument_list),

    member_expression: $ => prec(11, seq(field('object', $.expression), '.', field('name', $.identifier))),

    // Index (commas allowed: `m[a, b]` is the [] form of a macro call, shaped like an index)
    index_expression: $ => prec(11, seq(field('object', $.expression), '[', commaSep1(field('index', $.expression)), ']')),

    // space::fn(args) / T::static(args)
    scope_call: $ => prec(11, seq(
      field('scope', $.identifier), '::', field('name', $.identifier),
    )),

    // .{x: 1, y: 2}
    struct_literal: $ => seq('.', '{', optional(commaSep1($.struct_field)), '}'),
    struct_field: $ => choice(
      seq(field('name', $.identifier), ':', field('value', $.expression)),
      field('value', $.expression),
    ),

    list_literal: $ => seq('[', optional(commaSep1($.expression)), ']'),

    // new <type>[size]
    new_expression: $ => choice(
      // [size] must stick to the type (token.immediate): separates "sized heap allocation" from index syntax at the lexer level
      seq('new', field('type', $.type), token.immediate('['), field('size', $.expression), ']'),
      seq('new', field('type', $.type)),
    ),

    // ---- Types ----
    type: $ => prec.right(2, choice(
      // Bare pointer (an opaque FFI handle) and pointer <builtin type>;
      // a nullable reference to a custom type is written T& by corpus convention (avoids ambiguity with "bare pointer + variable name")
      seq('pointer', $.builtin_type),
      'pointer',
      $.anonymous_struct_type,
      $.anonymous_interface_type,
      seq($.type_base, $.type_arguments),
      seq($.type_base, '&'),
      seq($.type_base, '[', 'Copyd', ']'),
      $.interface_type,
      $.type_base,
    )),

    type_base: $ => choice($.identifier, $.builtin_type),

    type_arguments: $ => prec(3, seq('<', commaSep1($.type), '>')),

    // interface{} / interface{ } (empty interface = void)
    interface_type: _ => seq('interface', '{', '}'),

    // struct { int a; String s; } (anonymous struct as a type annotation)
    anonymous_struct_type: $ => prec(1, seq(
      'struct', '{', repeat($.member_declaration), '}',
    )),

    // interface { ... } / interface{ }
    anonymous_interface_type: $ => prec(1, seq(
      'interface', '{', repeat(choice($.method_signature, $.expand_declaration)), '}',
    )),

    builtin_type: _ => choice(
      'int', 'long', 'char', 'float', 'bool', 'String', 'void', 'Self',
      'List', 'HashTable', 'IOStream', 'thread', 'memorize', 'memory', 'function',
    ),

    // ---- Lexing ----
    identifier: _ => /[A-Za-z_\p{L}][A-Za-z0-9_\p{L}]*/u,
    number: _ => {
      const decimal = /[0-9]+/;
      const float = seq(decimal, '.', decimal);
      return token(choice(float, decimal));
    },
    string: _ => token(seq('"', repeat(choice(/[^"\\\n]/, seq('\\', /./))), '"')),
    raw_string: _ => token(seq('`', repeat(/[^`]/), '`')),
    boolean: _ => choice('true', 'false'),
    null: _ => 'null',

    line_comment: _ => token(seq('//', /[^\n]*/)),
    block_comment: _ => token(seq('/*', /[^*]*\*+([^/*][^*]*\*+)*/, '/')),
  },
});

function commaSep1(rule) {
  return seq(rule, repeat(seq(',', rule)));
}

function commaSep(rule) {
  return optional(commaSep1(rule));
}
