# QuarkLang v2 decided model (summary of 2026-08-30, archived)

**English** · [简体中文](v2-定案模型.zh-CN.md)

> This document collects all recent design decisions and was the basis for the full rewrite. Items marked ⚠ were
> still open at the time.
>
> **Historical archive (2026-08-30)**: the syntax has since been unified into the **canonical type-first form**;
> the examples below were rewritten accordingly. The authoritative syntax checklist is the main workspace's
> `SYNTAX.md`, and the current specification is `spec.md`.

## 1. Type system (FuncBuffer removed entirely)

- `void` = the **default name** of the empty interface `interface{}` (the language predeclares
  `interface {} void;` — "not empty, but no interface"). You may simply write `void` for the empty interface.
- `interface { ... }`: an **anonymous interface** (usable as a type annotation);
  `type interface { ... } Name;`: a named interface (`type` is required).
- `struct { ... }`: an **anonymous struct type**; `type struct { ... } Name;`: a named struct (`type` required).
- Struct literals: `.{field: value, ...}` (field names may be keywords such as `in`/`out`).
- `impl { ... } Type;`: the **only form** (interfaces are satisfied structurally; `impl` never names an interface).
- Generics `type struct<T> { ... } Box;` / `impl<T> { ... } Box;` (an impl must introduce the struct's generic
  parameters); pointers `T&` and `null` (automatic dereference); `Copyd<T>` (copy on pass, `.ptr()` unwraps). All retained.
- `List<T>`: a rolling two-cursor buffer (`*` takes the head, `next()`, exhaustion at `head()==tail()` raises,
  `reset()` rewinds).

## 2. Functions

- A **return type is required**: `fn f(int n) int { ... }` (main may omit it = void).
- `return expr;`: returns and ends the function.
- `log expr;`: records a log entry and ends the function (any value, void included).
- **No out parameters, no multiple return values**; multiple outputs use `List<T>` as the single return value.
  Calls evaluate directly to a value.

## 3. Error handling

- `try { ... } catch (<type> <name>) { ... }`: the catch must declare both type and name (e.g. `catch (void e)`);
  the error information is bound to `e`.

## 4. Signatures (Sign)

- `f(args) @instance(prefix)` ≡ `instance.call(prefix)(.{in: ..., out: ...})` (any Sign instance variable name):
  - `instance` is a Sign instance with any name (`ClassName memo = ClassName::new();`; a class with all the
    methods satisfies the Sign interface — structural satisfaction, no interface name on `impl`);
  - `instance.call(prefix)` yields a **function** that receives a `. {in, out}` two-field record and returns a result;
  - `memorize` memoizes by in→out.
- ⚠ How the function returned by `call` executes the wrapped original `f` (where the reference to `f` comes from) — open.

## 5. Tasks (the `taskm` global)

- `taskm.spawn()`: no arguments; creates a thread and returns its pid.
- `taskm.merge(pid, fn, args...)`: runs a function on thread `pid`.
- `taskm.block(pid)`: returns void (the empty interface) — it only waits; results travel through channels.
- `taskm.done(pid)`: whether that thread is idle (no function occupying it).
- `taskm.channel([n])`: default capacity 1024; IO execution is FIFO with reads first.

## 6. Memory

- Block allocation (`memory.setBlock(n)` sets the granularity), dirty marking, automatic reclaim marking when a
  task ends, and `compact()` for the actual cleanup (which does not return).

## 7. Macro system (named-parameter macros, decided)

- Definition: `#macro name (p1, p2, ...) { body }` — parameters inside `()`, comma-separated, any count; the
  delimiter for both the parameter list and the body may be `()`, `[]` or `{}` interchangeably.
- Invocation: `name(args)` / `name[args]` / `name{args}` — mirroring the definition; the argument count must
  match the parameters; parameter names inside the body are replaced by the argument tokens (single pass, no recursion).
- Dynamic preprocessing inside the body: `#when (compile|run) { ... }` selects a block by state;
  `#error("...")` reports an error.
- The legacy pattern syntax (`macro {pattern}{body}` plus `#insert/#ast/#execute`) is abolished.
- Predefined macros retained: `program main;` / `program library;` / `pub` / `import ...;`.

## 8. Open questions

1. ⚠ How a signature `call` executes the wrapped function `f`.
2. Is the macro system part of this v2 rewrite, or does v2 core land first with macros in a following round?
