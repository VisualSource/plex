---
name: 0036-linear-memory-strings
description: Lesson 36 — memory and data sections, string literals as compile-time-placed pointers; section-order constraint
metadata:
  type: project
---

# Linear Memory and String Literals — Binary Backend

User completed lesson 36. `TestEndToEnd_StringLiteral` passes after two encoder fixes (see below).

Concepts internalised:

- **Linear memory is a single byte array, page-sized in 64 KiB chunks.** WASM modules declare it via the memory section (id 5) with a min/max in pages; the runtime allocates and exposes it through `mod.Memory()` once exported. Strings live there, not on the value stack — the stack only ever carries the i32 pointer.

- **String layout = 4-byte LE length prefix + raw bytes.** Same shape as the WAT compiler. The "value" of a string literal at the WASM level is the i32 address of the length prefix; reading the string is "load 4 bytes for n, then read n more bytes". This makes equality of identical pooled literals a pointer compare.

- **Data section preloads bytes at instantiation.** Like `.rodata` in a native binary — the host writes the bytes into linear memory once when the module is instantiated, and code just reads them. Zero-cost at runtime: no allocator, no copy.

- **Offsets are encoded as constant expressions, not raw integers.** Each data segment's offset is a tiny instruction stream — `OpI32Const, <leb>, OpEnd` — terminated by `end`. Same shape as global initialisers and element segments. Generalises to e.g. `global.get`-based offsets without redesign.

- **Pooling literals by value.** A `stringTable` keyed by string content dedupes identical literals to a single offset, saving binary size and making string-identity coincide with string-equality.

## Section order is mandatory

This is the biggest takeaway from the lesson. The spec defines a fixed section order: type (1), import (2), function (3), table (4), memory (5), global (6), export (7), start (8), element (9), code (10), data (11). Validators reject out-of-order sections — the format is designed to be single-pass parseable. Memory **must** come before export (because export references it); data **must** come last (it references memory, code, and globals).

The first attempt emitted memory/data *after* code, after export. Wazero rejected with "invalid section order". The fix was reorder, not patch.

**Mental model:** sections aren't independent — they form a dependency graph. The numeric IDs encode a topological sort.

## The self-shadowing typo

A subtle bug worth flagging for future codegen: in `encodeDataSection`, `bytes` (the per-segment bytes vector) and `body` (the section body being accumulated) are easy to swap. Writing `body = append(body, body...)` instead of `body = append(body, bytes...)` doubles the section body, declared length stays correct (it's computed from `len(bytes)`), and the validator dies trying to read more bytes than the segment actually contains: "read bytes for init: unexpected EOF".

**Class of bug:** *correct length, wrong contents.* These are nasty because the binary "looks right" at the byte-count level but the contents are nonsense. A `wasm-objdump` round-trip would catch them instantly — worth wiring up that tool once memory work intensifies.

## Threading discipline

The string table joins `funcIndices` as a piece of module-level state threaded into every `bodyEncoder` via `newBodyEncoder`. Two pieces of read-only context now flow alongside each function body. Adding more (e.g. a globals table, a struct registry) follows the same shape — no new mechanism needed.

**State after lesson 36:** string literals work end-to-end as static memory. Lesson 36 also pre-added `OpI32Load (0x28)` and `OpI32Store (0x36)` opcodes, unused so far. The natural next lesson uses them: `s.len()` is a single `i32.load` at the string's pointer, then promote to i64.
