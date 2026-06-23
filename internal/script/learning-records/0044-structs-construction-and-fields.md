---
name: 0044-structs-construction-and-fields
description: Lesson 44 — struct constructors, field read (bare MemberAccess), field write (MemberAssignment); layout table threaded into bodyEncoder
metadata:
  type: project
---

# Structs — Construction, Field Read, Field Write

User completed lesson 44. `TestEndToEnd_StructFields` passes — `Point(3.0, 4.0)`, `p.x`, `p.x * p.x + p.y * p.y`, and `p.y = 99.0` all work. Full suite green (21 tests).

Concepts internalised:

- **Struct memory has no header.** Fields are packed in declaration order at compile-time byte offsets. The struct's runtime value is the i32 pointer to the first field. Compare: arrays have a 4-byte length prefix because length is *runtime-known*; structs don't because layout is *compile-time-known*. The header is data the runtime needs that the compiler couldn't produce ahead of time — that's the heuristic.

- **The layout table is module-level state.** A `map[string]*structLayout` is built once during `CompileProgram` (via `collectStructs`) and threaded into every `bodyEncoder` alongside `funcIndices` and `strings`. Same threading pattern, third instance — module-level read-only state shared with every function body.

- **`Point(3.0, 4.0)` is a `FunctionCall`, not a dedicated literal node.** The disambiguation happens at codegen: check `b.structs[name]` *before* `b.funcIndices[name]`. The parser stays untouched; one new branch in the FunctionCall.Identifier case routes to a constructor helper.

- **Bare `*script.MemberAccess` is new in the walker.** Until today it only appeared inside `FunctionCall.Callee` (lesson 37's `s.len()`). Today it appears in expression position (`p.x`). A new top-level case in `walk` handles it: look up the field offset, emit `typed.load offset=<field.offset>`. No `+4` skip, no index arithmetic — field offsets are compile-time constants.

- **`MemberAssignment` was already plumbed.** AST node and typechecker arm existed since the WAT compiler — only the binary-backend codegen case was missing. Same shape as field read but terminates in `typed.store`.

## Three allocator variants, one shape

The binary backend now has three heap-allocation paths, all following the same skeleton:

| Lesson | Construct | Header | Per-element/field offset |
|--------|-----------|--------|--------------------------|
| 36 | String literal | 4-byte length | (none — bytes follow) |
| 40 | Array literal | 4-byte length | `4 + i * elemSize` |
| 44 | Struct literal | (none) | `layout.fields[i].offset` |

**The skeleton**: bump-allocate N bytes via `__heap_ptr`, populate via per-element memarg offsets, leave base pointer on the stack. The variation is in: header presence, offset computation, store typing. A `(b *bodyEncoder).emitHeapInit(...)` helper would consume all three call sites — pin for after the fourth call site appears (method receivers? closures?). Premature factoring at three sites is still premature.

## Implementation

- `structLayout` and `structField` types in `structs.go`.
- `collectStructs(p *Program) map[string]*structLayout` in helpers.go — walks `StructStatement` declarations and assigns sequential offsets via `elemSizeOf`.
- `bodyEncoder` gains `structs map[string]*structLayout` field, threaded through `newBodyEncoder`, `encodeCodeEntry`, `encodeCodeSection` from `module.go`.
- `programHasHeapAllocation(p, structs)` widens the heap gate to include struct construction.
- `walkStructConstructor` mirrors the array allocator: bump prologue → per-field store at `field.offset` → leave base pointer.
- New `*script.MemberAccess` top-level case for `p.x` reads.
- New `*script.MemberAssignment` case for `p.x = v` writes.

## Future improvements pinned

- **Field alignment padding.** Current layout is packed (no padding). For `{ a: i32; b: f64; }` this puts `b` at offset 4 — 4-byte aligned, not 8. WASM accepts unaligned loads, but some engines optimise the aligned path. Real compilers add per-field padding before placing each field: `offset = (offset + (size-1)) & ~(size-1)`. Two-line change in `collectStructs`; do it when performance matters.
- **Heap pointer alignment.** Same story at the heap level: `strings.next` might land mid-byte. `(strings.next + 7) & ~7` rounds the start up to 8.

**State after lesson 44:** the binary backend covers strings, arrays, structs, all primitive operations, control flow, and function calls. The next language gap is methods — `impl Block { fn m() {...} }`. The infrastructure is mostly there: `collectSignatures` already emits sigs for impl methods with mangled names (`__Struct__method`) and an implicit `self: i32` first param. `newBodyEncoder` already wires `self` into `b.locals[0]`. What's missing is the call-site dispatch: when `FunctionCall.Callee` is a `MemberAccess` on a struct object, push the receiver, push args, call by mangled name.
