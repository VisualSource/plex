---
name: 0043-bounds-checks-and-unreachable
description: Lesson 43 — array bounds checking via i32.ge_u + unreachable; WASM decoder errors point at symptom not cause
metadata:
  type: project
---

# Bounds Checks and the Unreachable Trap — Binary Backend

User completed lesson 43. `TestEndToEnd_BoundsInBounds` and `TestEndToEnd_BoundsOutOfRange` pass. Out-of-bounds reads and writes now trap loudly instead of reading garbage. Full suite green.

Concepts internalised:

- **WASM gives static type-safety but not static memory-safety for typed loads/stores.** Any in-page address validates; the engine doesn't know which addresses are "inside an array" and which are "between arrays". Languages that want bounds-safety have to emit the checks themselves. Lesson 43 made plex one of those languages.

- **`unreachable` (0x00) is the universal WASM trap.** No exception mechanism, no payload, no host-call infrastructure needed — just a single byte that aborts the module with an error wazero surfaces from `Call`. You'd been emitting it since lesson 28 as a defensive cap after every function's `return`, but until today it lived only in unreachable code paths. Today's use is the first reachable one.

- **Unsigned compare catches negative indices for free.** `i32.wrap_i64` of a negative i64 yields a value with the high bit set — read as unsigned, it's a huge u32, always > any reasonable array length. `i32.ge_u` (0x4F) covers both "too big" and "negative" in one comparison.

- **The synthetic-locals pattern repeats.** Same allocator-tmp shape from lesson 40: stash the base and the wrapped index in two synthetic locals (each declared via `addSyntheticLocal`), then reload them as needed for the bounds check and the address arithmetic. WASM lacks a `dup` instruction, so locals are the standard reuse mechanism.

## The decoder-error lesson

The error from the broken implementation was: `read block: type index out of range: 33`. Decoded:
- `0x21` = `OpLocalSet` = 33 in decimal
- "type index" means the validator is reading a block-type field
- Block-type fields appear after `OpBlock`/`OpLoop`/`OpIf` (opcodes 0x02/0x03/0x04)
- So the parser believed it was just past `OpBlock` (`0x02`)
- The byte that *was* `0x02` was a stray `arrIdx` ULEB128 emitted right after `i32.wrap_i64` — which takes no operand

The cascade: one extra byte emitted upstream knocked the parser off its rails, and every subsequent byte was misinterpreted. The error fired many instructions downstream of where the bug actually lived.

**Pinned principle**: **WASM decoder errors point at the symptom site, not the cause site.** The cause is whatever upstream byte first knocked the parser off its rails. When chasing one, scan backwards from the failure looking for the *first* instruction that emitted a byte the parser couldn't have expected.

Concrete diagnostic discipline: if the validator complains about an opcode/operand combination that *should* be valid, suspect a stray byte upstream — usually a forgotten/extra ULEB128 or a missing block-type byte.

## Implementation

- `OpI32GeU = 0x4F` added.
- `*script.ArrayAccess`: stash base + wrapped idx into synthetic locals; load length; compare `idx >= len` with `i32.ge_u`; `if (block_type_empty); unreachable; end`; address arithmetic; typed load.
- `*script.ArrayAssignment`: same prologue, then push value, then typed store.

## Cost and the optimisation horizon

About nine extra instructions per indexed access. Real cost for tight loops. **Bounds-check elimination** (BCE) is the mature optimisation that earns this back — Rust's iterators, Go's compiler, .NET's `Span<T>` all rely on it. Plex doesn't have BCE, but the shape of the code emitted today is exactly what such a pass would later eat. Worth noting that "emit checks now, optimise them out later" is the canonical approach, not "skip checks for speed".

**State after lesson 43:** arrays are now safe to index. The remaining language-feature gaps in the binary backend are structs (literals, field access, field assignment) and methods. Structs are the next obvious step; methods can layer on top.
