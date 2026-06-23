---
name: 0040-array-literals-bump-allocator
description: Lesson 40 — inline bump allocator at array-literal sites; memarg offset as displacement; synthetic locals; locals-vec finalisation order
metadata:
  type: project
---

# Array Literals and the Inline Bump Allocator — Binary Backend

User completed lesson 40. `TestEndToEnd_ArrayLiteral` passes after two structural fixes (see below). Full suite green (16 tests).

Concepts internalised:

- **Array layout = string layout, but typed.** 4-byte LE length prefix + N × elemSize bytes. The array's runtime "value" is the i32 address of the length prefix, exactly like strings.

- **The bump allocator is six inline instructions.** `global.get → local.tee tmp → i32.const N → i32.add → global.set` advances the heap pointer and stashes the base in a synthetic i32 local. Each array-construction site embeds its own prologue — no separate alloc function. Cheap if arrays are rare, code-bloaty if they're everywhere; the WAT compiler made the opposite choice (a single `$alloc` function emitted from a runtime helper).

- **memarg's `offset` field is the array-store/load arithmetic.** A store does `addr + offset` as part of its semantics. Three element stores produce zero `i32.add` bytecode — each store carries its own constant displacement (`offset=4`, `offset=12`, `offset=20`). Lesson 37 showed this for a single load; arrays exploit its real power.

- **Synthetic locals are normal — but they have a subtle counter requirement.** `b.localTypes` is just a vec; appending creates a new local. But `len(b.locals)` is *also* the next-free-index counter — so synthetic locals need their `len()` to advance, otherwise multiple synthetics collide on the same index. Placeholder names (`__synth_<idx>`) keep the counter honest; source code can't reference them since plex identifiers don't start with `__`. The lesson originally claimed synthetics "don't need an entry in b.locals" — that conflated the two purposes of the map (name lookup vs. index allocation) and was wrong on the second.

## Two structural bugs caught and pinned

### Section-emission gate widened
The `needsHeap` flag correctly added the memory + `__heap_ptr` exports for arrays-only programs, but the *section emission* still checked `len(strings.entries) > 0`. So the exports referenced sections that weren't there → "memory for export[memory] out of range" at instantiation. Fix: gate memory + global section emission on `needsHeap`; data section stays gated on strings (no strings → no data segments).

### Locals vec finalisation order
`encodeCodeEntry` wrote the locals vec *before* calling `walk` — but synthetic locals are appended *during* walk (when an `ArrayLiteral` is encountered). So the locals declaration was empty when the validator checked, and `local.tee 0` blew up with "invalid local index 0 >= 0(=len(locals)+len(parameters))".

**Fix and general principle**: walk first, then write `[locals_vec_count][locals_entries]`, then append the buffered body bytes. **The locals declaration must be finalised AFTER body emission, even though it's serialised BEFORE the body in the section.** Every future codegen-emitted local (struct tmps, spill slots, allocator scratch) needs this same ordering. Pin this as a permanent discipline.

## Implementation

- `OpI64Store = 0x37`, `OpF64Store = 0x39` added (i32.store was already present from lesson 36).
- `programHasArrays(p *Program) bool` AST walker that mirrors `collectStrings`'s shape — same collector-emitter invariant as lesson 37.
- `elemSizeOf(TypeKind) uint32` and `storeOpcode(TypeKind) (op, align)` helpers in helpers.go.
- `addSyntheticLocal(vt byte) uint32` on `bodyEncoder`, with `__synth_<idx>` placeholder names in `b.locals` to keep the index counter advancing.
- `*script.ArrayLiteral` case in walk: alloc prologue → store length at offset 0 → store each element at offset 4+i*elemSize via memarg → leave base pointer on stack for the parent.
- `module.go`'s `needsHeap` flag gates both the export entries AND the section emission.
- `encodeCodeEntry` reordered: walk → write locals vec → append body bytes.

**State after lesson 40:** arrays are constructable from source. The natural next step is using them — `arr.len()` (mechanically identical to `s.len()` from lesson 37) and `arr[i]` (which needs `i32.wrap_i64` for the index, plus a typed load with `offset=4` to skip the length prefix).
