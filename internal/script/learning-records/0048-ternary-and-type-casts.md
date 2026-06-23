---
name: 0048-ternary-and-type-casts
description: Lesson 48 — ternary expressions (if-with-result block type) and type cast opcodes in binary backend
metadata:
  type: project
---

# Ternary Expressions and Type Casts

User completed lesson 48. `TestEndToEnd_Ternary` and `TestEndToEnd_TypeCast` pass. Full suite green (25 tests).

## Ternary = if with a value-type block type

The ternary `cond ? a : b` emits `OpIf <valType(result)>` instead of `OpIf BlockTypeEmpty`. The valtype byte (0x7E for i64, 0x7C for f64, 0x7F for i32) tells the WASM validator that both branches must leave exactly one value of that type on the stack. `blockDepth` is not incremented — ternary is an expression, not a statement, and cannot contain `break`/`continue`.

## Cast opcodes added

Four new constants in `opcodes.go`: `OpF64ConvertI64S` (0xB9), `OpF64ConvertI32S` (0xB7), `OpI64TruncF64S` (0xB0), `OpI32TruncF64S` (0xAA). Plus `normKind` to collapse int→i64 / float→f64 aliases before table lookup, and `castOpcode(from, to)` returning `(0, true)` for same-type no-ops.

## State after lesson 48

The binary backend now handles every expression form the text backend supports. Remaining work (if any) would be import/export surface features or runtime helpers.
