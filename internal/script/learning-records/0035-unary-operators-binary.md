---
name: 0035-unary-operators-binary
description: Lesson 35 — unary - and + in the binary backend; WASM's missing integer negate composed from const 0 and sub
metadata:
  type: project
---

# Unary Operators — Binary Backend

User completed lesson 35. `TestEndToEnd_UnaryMinus` passes; the full suite still passes.

Concepts internalised:

- **WASM has no integer negation opcode.** `i32.neg` and `i64.neg` simply do not exist in the spec. Floats get `f64.neg` (`0x9A`) because it is a sign-bit flip — distinct from subtraction at the hardware level. Integer negation is exactly `0 - x` on a two's complement machine, so WASM offers no shortcut. The compiler composes: emit `i64.const 0`, walk the operand, emit `i64.sub`. The push order matters — push 0 first, then operand, because `sub` pops the top as the right operand.

- **Unary `+` is a syntactic placeholder.** No opcode is emitted; the parser produces a `UnaryExpression` node only because the grammar allows the syntax, but at codegen time the unary `+` short-circuits to walking the operand directly.

- **Operand-vs-result type, again.** Like lesson 34, dispatch on `n.Operand.GetType()`, not `n.GetType()`. For negation the two coincide (negation preserves type), but the rule is invariant: the type that selects the opcode is the type going *in*.

Implementation:
- `OpF64NEg` (their naming) added to opcodes.go at `0x9A`.
- `*script.UnaryExpression` case in body.go handles `TokenType_Plus` (walk operand only) and `TokenType_Minus` (f64/i64 paths).
- No i32 path yet — fine, no test exercises it. Easy to add when needed by mirroring the i64 path with `OpI32Const` / `OpI32Sub`.
- No `!` because plex's parser doesn't produce it.

**State after lesson 35:** the binary backend now covers every primitive-typed AST node plex's parser produces — params, locals, literals, arithmetic, comparisons, AND/OR (short-circuit), if/while, break/continue, return, function calls, drop, unary +/-. The next layer is linear memory: strings, arrays, structs all require it.
