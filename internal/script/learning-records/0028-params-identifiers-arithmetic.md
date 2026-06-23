---
name: 0028-params-identifiers-arithmetic
description: Lesson 28 — params, local.get, and binary arithmetic in the binary WASM backend
metadata:
  type: project
---

# Params, Identifiers & Binary Arithmetic — Binary Backend Body Encoder

User completed lesson 28. The `bodyEncoder.walk` switch now handles `*Identifier` and `*BinaryExpression`, enabling any function whose body is pure arithmetic over its parameters to be compiled to valid WASM binary.

Concepts internalised:

- **Params share the locals index space.** `local.get` (`0x20`) takes a single u32 index: params come first (0..n-1), future `let` locals follow from index n onward. `newBodyEncoder` pre-populates `b.locals` from the function signature so `*Identifier` can look up the index at emit-time.

- **WASM is a stack machine.** A binary expression `a + b` compiles to: push left → push right → emit arith opcode. `walk(n.Left)` then `walk(n.Right)` naturally respects left-to-right order; the test `sub3(100, 50, 8) → 42` caught a walk-order bug (expected 42, would get 58 if Right was walked first).

- **Arithmetic opcodes are type-specific.** `arithOpcode(TypeKind, Operator)` dispatches to i32/i64/f64 variants. This table pattern will repeat for comparison, bitwise, and conversion ops later.

- **Latent bug identified but deferred:** `OpI64Mul byte = 0x7F` should be `0x7E` — it currently collides with `OpI64DivS = 0x7F`. No i64 multiplication test exists yet so this has gone unnoticed. Fix is in lesson 29 (while opcodes.go is open to add `OpLocalSet`).

**State after lesson 28:** params + constants + arithmetic work end-to-end. Next blocker: `*VariableDeclaration` is unhandled, and `encodeCodeEntry` hardcodes `0` for the locals vec.
