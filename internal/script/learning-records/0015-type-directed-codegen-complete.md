---
name: type-directed-codegen-complete
description: Lesson 15 complete — array element type-aware store/load, snapshot tests fixed, all compiler tests green
metadata:
  type: project
---

All four compiler tests pass. The user:
- Fixed `ArrayLiteral` to derive element wasm type from `n.GetType().Element` instead of hardcoding `f64`
- Fixed `ArrayAccess` to use type-aware load and stride
- Created `testdata/alloc.wat` and `testdata/array.wat` for the missing snapshots
- Fixed `TestCompilerFunctionCall` source (wrong `int` return type → `float`)
- Fixed `TestStructGenAndCtor` snapshot (trailing newline mismatch)

**Why:** The typechecker annotates the AST; the compiler must read those annotations. Hardcoding `f64.store` for any array element produces invalid WAT when the element type is `i64`.

**How to apply:** Any time the compiler emits a typed instruction (store, load, const, add), check whether it should derive that type from `getWasmType(n.GetType())` rather than hardcoding a string.
