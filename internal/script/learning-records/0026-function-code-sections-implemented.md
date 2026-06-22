# Function + Code Sections Implemented

User completed lesson 26 and now has all the machinery for declaring and bodying functions in the binary backend:

- The **Function section** (id 3) — a `vec(typeidx)`. With no signature dedup, function `i`'s typeidx is just `i`, so the body is literally `[count, 0, 1, 2, …]`.
- The **Code section** (id 10) — `vec(code)` where each entry is doubly length-prefixed (per-entry size + section size). Each entry contains a `vec((count, valtype))` for locals followed by the body's opcode bytes and a trailing `0x0B end`.
- Six opcodes in [opcodes.go](../compiler/binary/opcodes.go): `end` (0x0B), `return` (0x0F), `local.get` (0x20), `i32.const` (0x41), `i64.const` (0x42), `f64.const` (0x44).
- A `bodyEncoder.walk` in [body.go](../compiler/binary/body.go) that recursively translates `*Block`, `*NumberLiteral`, `*ReturnStatement` to bytes. The `default` case errors loudly with the AST node type, which is the trigger for adding new opcodes lesson-by-lesson.
- **Critical encoding distinction internalised:** integer `*.const` immediates are signed LEB128; `f64.const` immediate is 8 raw little-endian IEEE-754 bytes (no LEB128).

Also cleaned up two carry-over bugs from lesson 25: `SectionMemory` was 4 (now correctly 5, with `SectionTable=4` declared), and the `implName == ""` check in `signatureOf` was inverted (now `!= ""`).

**Implications:** The user can produce a `.wasm` with Type + Function + Code, and `wasm-objdump -d` correctly disassembles the bodies. The file still isn't *runnable* — no Export section means a host has no callable handle into the module. Lesson 27 adds Export and the binary backend produces its first runnable end-to-end module. After that, lesson 28 expands the opcode set (binary operators, locals, function calls).

Two latent bugs to fix as part of lesson 27, before they cascade:
- [body.go:43](../compiler/binary/body.go#L43) — `binary.LittleEndian.AppendUint64(raw[:], …)` silently zeroes every f64 constant (Append returns a new slice; the array's bytes never change). Must be `PutUint64`.
- [helpers.go:24](../compiler/binary/helpers.go#L24) — methods still pass `m.Name` (method name) as `implName`. Bug is dormant because `implName != ""` is true either way, but the emitted symbol becomes `__methodName__methodName` instead of `__StructName__methodName`. Will matter the moment we look up methods by their mangled name.
