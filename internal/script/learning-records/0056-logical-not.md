---
name: 0056-logical-not
description: Lesson 56 — adding ! (logical NOT) across all compiler stages; token disambiguation; i32.eqz; the full-stack operator template
metadata:
  type: project
---

# Logical NOT (`!`)

Lesson 56 adds `!` as a prefix unary operator to plex. It's the minimal full-stack feature: one new token type, one new check in the typechecker, and one new opcode emit in each codegen backend.

## Current state

`!` is not a token. The tokenizer handles `!=` but falls through to `ErrUnexpectedCharacter` when `!` is not followed by `=`. Writing `!x` today is a tokenizer error.

## Operator precedence

`!` is a prefix operator: tighter than all binary operators, same level as unary `-` and `+`. It belongs in `parseUnary()` in ast.go, alongside the existing `-`/`+` handling.

## Token disambiguation

The tokenizer peeks one character ahead when it sees `!`:
- Next char is `=` → emit `TokenType_NotEqual`, advance past `=`
- Next char is anything else → emit `TokenType_Not` (new), stay in place

This is the same lookahead-1 pattern used for `==`, `<=`, `&&`, `||`.

## Files changed

| File | Change |
|---|---|
| tokens.go | Add `TokenType_Not` to iota, `String()`, and `tokenSymbol` |
| tokenizer.go | Replace `fallthrough` with emit of `TokenType_Not` + `continue` |
| ast.go | Add `TokenType_Not` to `parseUnary()` condition |
| typeschecker/typecheck.go | UnaryExpression: require bool operand, return bool |
| compiler/compiler.go | WAT codegen: emit `i32.eqz` for `TokenType_Not` |
| compiler/binary/body.go | Binary codegen: append `OpI32Eqz` for `TokenType_Not` |

## The WASM opcode

`i32.eqz` (0x45) — already in opcodes.go. Pops one i32, pushes 1 if it was 0, pushes 0 otherwise. Since plex booleans are i32 (false=0, true=1), this is exactly logical NOT.

## Typechecker invariant

`!bool → bool`. The typechecker rejects `!42` (int operand) before codegen runs. The existing UnaryExpression handler propagates the operand type unchanged — `!` is the first case that needs to both validate the operand type and return a different (fixed) result type.

## Tests added

`TestEndToEnd_Not`:
- `!true` → 0, `!false` → 1 (simple cases)
- `!!x` → x (double negation, cancels out)
- `!(a == b)` → inverted comparison result
