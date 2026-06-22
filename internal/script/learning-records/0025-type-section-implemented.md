# Type Section Implemented

User completed lesson 25. They built the foundational primitives every later section reuses:

- The **section wrapper** pattern (`id + uleb128(size) + body`) in [section.go](../compiler/binary/section.go), used by every section emitter.
- The **`vec(T)` primitive** (a uleb128 count + N elements) — central to the spec's notation.
- The **valtype byte encoding** (`0x7F/7E/7D/7C` for i32/i64/f32/f64) and how plex's broader type system maps onto it (all reference-like types — bool, string, array, struct — collapse to i32).
- The **functype encoding** (`0x60` + vec(params) + vec(results)) and that a void return is an empty vec, not an absent one.

The Type section now appears in the emitted `.wasm` and `wasm-objdump -x` shows one entry per declared plex function (top-level + methods). The package is named `binary_wasm` (to avoid colliding with the stdlib `encoding/binary` import). No deduplication of identical signatures yet — each function gets its own type entry in declaration order; this 1:1 mapping makes the Function section's typeidx encoding trivial in lesson 26.

**Implications:** The user now thinks in terms of "everything is a vec" and "every section is wrapped." They're ready for the Function + Code sections, which are tightly coupled (the spec requires their counts to agree for the module to validate). Lesson 26 introduces the first opcodes (`end`, `return`, `i*.const`) and the Code section anatomy.
