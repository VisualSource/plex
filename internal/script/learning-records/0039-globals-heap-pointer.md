---
name: 0039-globals-heap-pointer
description: Lesson 39 — global section, mutable __heap_ptr initialised to strings.next; export for host inspection
metadata:
  type: project
---

# Globals and the Heap Pointer — Binary Backend

User completed lesson 39. The module now emits a global section between memory (5) and export (7) whenever it emits memory at all. `__heap_ptr` is mutable i32, initialised to `stringTable.next` (the byte right after the last string literal), and exported by name.

Concepts internalised:

- **WASM globals are just typed module-level slots.** `{type, mutability, init expr}` is the whole structure. Mutability is a single byte (0=const, 1=mut); type is a valtype byte; init is a constant-expression instruction stream terminated by `OpEnd`. The init form is *identical* to what data segment offsets use (lesson 36) — same three-byte shape `OpI32Const <leb> OpEnd`. One pattern, reused everywhere a compile-time constant is needed.

- **Section order, again.** Globals (6) live between memory (5) and export (7). Same single-pass-parseability constraint as lesson 36. Easy to get right because the rule is "ID order is byte order".

- **Module-level mutable state is an architectural step, not just a syntax addition.** Until this lesson the module had only function-local mutable state (params, locals). Globals let the module *remember* between calls — the structural prerequisite for any allocator, instance counter, lazy-init cache, or GC. Lesson 39 spends none of that capability; lesson 40 will.

- **`strings.next` is exactly the heap start.** The string table's bookkeeping (`add` bumps `next` by `4 + len(value)`) makes the heap-pointer initialiser a one-liner. No second pass, no parallel state.

- **`local.tee` is the dup-after-set idiom** (foreshadowed for next lesson). Plain `local.set` consumes the value; `local.tee` writes to the local *and* leaves the value on the stack. Useful when you need a value once on the stack and again from a named slot — the array-literal alloc sequence will lean on it.

## Implementation

- `OpGlobalGet = 0x23`, `OpGlobalSet = 0x24` added to opcodes.go (unused this lesson; ready for next).
- `encodeGlobalSection(heapStart uint32)` emits one mutable i32 global with the init expression `i32.const heapStart; end`.
- `module.go` emits the global section right after the memory section, gated on `len(strings.entries) > 0`, and registers `__heap_ptr` as an `ExportGlobal` export so the host can `mod.ExportedGlobal("__heap_ptr").Get()`.

## Tests pinned

- `TestEndToEnd_HeapPointer`: one string `"hello"` → 9 bytes consumed → heap starts at 9. Asserts the exported global reads back as 9.
- `TestEndToEnd_HeapPointerMultipleStrings`: `"hi"` (6 bytes) + `"world"` (9 bytes) = 15 bytes consumed → heap starts at 15. Pins the cumulative arithmetic.

**State after lesson 39:** the module gains its first piece of cross-call mutable state. No allocator yet — `__heap_ptr` is read-only-in-practice until lesson 40 writes the bump-allocator inline at array-construction sites.
