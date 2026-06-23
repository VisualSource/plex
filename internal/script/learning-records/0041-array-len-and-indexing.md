---
name: 0041-array-len-and-indexing
description: Lesson 41 — arr.len() reuses s.len() shape; arr[i] needs i32.wrap_i64 + memarg offset=4
metadata:
  type: project
---

# Array Length and Indexing — Binary Backend

User completed lesson 41. Arrays are now readable as well as constructable. `TestEndToEnd_ArrayLen` and `TestEndToEnd_ArrayIndex` pass; full suite green.

Concepts internalised:

- **`arr.len()` and `s.len()` are byte-identical.** Both compile to `i32.load align=2 offset=0` followed by `i64.extend_i32_s`. Not coincidence: strings (lesson 36) and arrays (lesson 40) deliberately share their length-prefix-at-offset-0 layout. Any future length-prefixed container (vec, slice, dynamic string) inherits this codegen for free. Tempting to factor the two arms into one ("anything with a header"); resist until a third sibling appears.

- **`arr[i]` is address-arithmetic plus a typed load.** The full sequence is `base, idx, wrap_i64, sz, mul, add, typed.load offset=4`. Four numeric facts conspire: base is i32 (an address), idx is i64 (plex's `Int` default), elemSize is a compile-time constant, and the `+4` length-prefix-skip rides in the memarg.

- **`i32.wrap_i64` is the inverse of `i64.extend_i32_s`.** They're the i32↔i64 type-bridging pair. Lesson 37's `s.len()` used extend (widening for the Int return type). Today's `arr[i]` uses wrap (narrowing for the i32 address arithmetic). The validator counts stack types strictly — without the wrap, `i32.mul` below an i64 fails validation.

- **memarg's `offset` field replaces an `i32.add 4`.** Same trick as the array stores in lesson 40. The displacement rides in the instruction, not on the stack. Loops over arrays now produce dense bytecode — no per-iteration "add 4" instruction.

## What's now possible end-to-end

```
fn total(): int {
    let arr = [1, 2, 3, 4, 5];
    let i: int = 0;
    let sum: int = 0;
    while i < arr.len() {
        sum = sum + arr[i];
        i = i + 1;
    }
    return sum;
}
```

Array construction, indexed reads, length-comparison-as-loop-bound — entirely from plex source, compiled to bytecode wazero runs. The binary backend has reached language-completeness for programs that don't mutate arrays.

## Implementation

- `OpI64Load = 0x29`, `OpF64Load = 0x2B`, `OpI32WrapI64 = 0xA7` added.
- `loadOpcode(TypeKind) (op, align)` helper mirroring `storeOpcode`.
- `*script.TypeKind_Array` arm added under `MemberAccess` in the `FunctionCall` dispatch — byte-for-byte identical to the string `len` arm.
- `*script.ArrayAccess` case in walk: walk target → walk index → `i32.wrap_i64` → `i32.const elemSize` → `i32.mul` → `i32.add` → element-typed load with `offset=4`.

## Limitations pinned for future lessons

- **No bounds check.** `arr[100]` on a 3-element array reads neighbouring bytes (or traps on cross-page access). Three instructions (`i32.lt_u idx, len; if; unreachable; end`) would fix it cheaply.
- **`arr[i] = x` doesn't parse.** Plex's `parseAssignment` (ast.go:500) only accepts `Identifier` and `MemberAccess` as assignment targets — `ArrayAccess` falls through to "invalid assignment target". Mutating array elements is a multi-stage lesson (parser + AST + typechecker + codegen).
- **String indexing semantics.** Typechecker reports `s[i]` as `string`, not a byte. The binary backend doesn't handle string indexing yet; aligning the semantics (return a byte? a 1-char substring?) is its own decision before codegen.

**State after lesson 41:** the read side of arrays is complete. The write side requires touching every compiler stage (the natural next lesson), or bounds-check + struct work as alternatives.
