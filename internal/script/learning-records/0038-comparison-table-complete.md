---
name: 0038-comparison-table-complete
description: Lesson 38 — i32 and f64 comparison opcodes complete; wazero result-buffer aliasing gotcha; literal-coercion limitation surfaced
metadata:
  type: project
---

# Comparison Table Complete — i32 + f64

User completed lesson 38. `TestEndToEnd_I32Comparisons` and `TestEndToEnd_F64Comparisons` pass; full suite green (14 tests).

Concepts internalised:

- **Signed/unsigned asymmetry between ints and floats.** Integer comparisons have `_s` and `_u` variants (`i32.lt_s` 0x48 vs `i32.lt_u` 0x49) because the same bit pattern answers "less than" differently per sign interpretation. Floats have only one variant per ordering op (`f64.lt` 0x63) — IEEE 754's sign bit is part of the encoding, not a label applied externally, so there is no "unsigned float" interpretation. Equality (`eq`/`ne`) doesn't need the integer split either, because bit-equality is interpretation-agnostic.

- **Plex's integers are signed.** All comparisons map to the `_s` variants. If plex ever grows a `u32`/`u64` type, that's a future `_u` arm — the cmpOpcode table is keyed by `(TypeKind, op)` so the extension is mechanical.

- **Floats produce i32 results.** `f64.lt` pops two f64s and pushes an i32 (0 or 1). The result is bool-shaped at the WASM level; the typechecker tags the AST node as `Bool` per lesson 34's fix.

## The bugs this lesson surfaced

### 1. wazero result-buffer aliasing (test infrastructure)

The single failure of `near_one(1.0)` reported `got[0] = 0x3FF0000000000001` — looks like `f64bits(1.0) + 1`. Diagnosis: wazero's `ExportedFunction.Call(ctx, args...)` reuses the args slice as the result buffer. For i32 returns (which includes bool), the engine writes only the low 32 bits; the high 32 bits carry forward whatever was in `args[0]`. With i64 args (all prior tests) the high bits were 0 anyway, so the bug never showed. With f64 args, the high bits are `0x3FF00000` (the exponent of 1.0), so the comparison `got[0] == 1` failed even though the function had computed 1 correctly.

**Fix**: `runOne` now consults `exportedFn.Definition().ResultTypes()` and masks `got[0] = uint64(uint32(got[0]))` when the declared result is `api.ValueTypeI32`. Preserves full i64 bits when the function actually returns i64 (e.g., the `uint64(-7)` case in lesson 35).

**Generalisable insight**: a "passing test" only tested what its inputs touched. As soon as test inputs covered new bit-pattern territory (f64 args), latent test-helper assumptions broke. Test helpers are part of the system under test.

### 2. Insufficient boundary tests in the table

I had `cmpOpcode` mapping `<` for i32 to `OpI32LeS` instead of `OpI32LtS` (typo: le_s vs lt_s). The i32 test passed anyway because every case used distinct operands (`lt(3,7)`, `lt(7,3)`), and at distinct operands `<` and `<=` agree. The boundary case `lt(3,3)` would have returned 1 (wrong) under the typo, 0 (right) under the fix.

**Testing principle pinned**: when adding a family of related opcodes (lt/le/gt/ge or any sibling set), at least one test per opcode must hit the boundary where its semantics diverge from its siblings. For comparisons, that boundary is `op(x, x)`.

### 3. Plex doesn't coerce integer literals to declared type

The original i32 test had `let i: i32 = 0;` and `i = i + 1;` and got rejected with "type annotation does not match init". The literal `0` always parses as `Int` (i64). The test was rewritten to use only parameters (which carry their declared type into scope), but this is a real plex limitation that will recur. Likely candidates for a future lesson:

- Coerce integer literals at the type-annotation site (so `let i: i32 = 0` works)
- Or add `0i32`-style suffixes
- Or default integer literals to `Int` and require explicit casts at i32 boundaries

For now, document the limitation: **inside an `i32` context, all numeric literals must come via a parameter or an explicit cast** (which the binary backend doesn't yet support).

## Implementation

- Added opcodes: `OpI32LtS = 0x48`, `OpI32GtS = 0x4A`, `OpI32LeS = 0x4C`, `OpI32GeS = 0x4E`; `OpF64Eq = 0x61` through `OpF64Ge = 0x66`.
- Extended `cmpOpcode`'s i32 arm with the four ordering ops; added a full f64 arm.
- Test infrastructure: `runOne` now masks i32 returns; new `f64bits` (or `api.EncodeF64`) helper for passing f64 args.

**State after lesson 38:** every primitive comparison plex's grammar produces compiles in the binary backend. The next architectural piece is the heap: arrays need a bump allocator, which needs a mutable global, which needs the global section and its constant-initializer encoding.
