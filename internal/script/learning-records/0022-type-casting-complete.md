# Type Casting with `as` — Full Pipeline Change

User implemented the `as` keyword across all five compiler stages in lesson 22: tokenizer → parser → AST node → typechecker → compiler. Key insight: the compiler must normalise `TypeKind_Int → i64` and `TypeKind_Float → f64` before looking up WAT conversion instructions, so that `x as int` and `x as i64` produce the same instruction. User also understands that `i64.trunc_f64_s` traps on NaN/overflow (by design) and why `i64 → f64` may lose precision (53-bit significand vs 63-bit value range).

**Implications:** User is comfortable tracing a feature end-to-end through the full compiler pipeline. Next sessions can introduce features without hand-holding on which files to touch — the pattern is now internalised.
