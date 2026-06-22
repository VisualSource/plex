# Export Section — First Runnable .wasm From the Binary Backend

User completed lesson 27. The binary backend now produces a `.wasm` that instantiates and runs end-to-end through wazero, with **zero** `wat2wasm` involvement.

Concepts internalised:
- **Exports** are the public surface of a module: (name, kind, idx) triples. The four kinds — func/table/memory/global — share a single export-descriptor encoding.
- **WASM has no string type at the binary level.** Anywhere a string appears — export names, future imports, custom-section payloads — it's `vec(byte)`: uleb128 length + UTF-8. Helper `encodeName` extracted in [section.go](../compiler/binary/section.go).
- **Section order is part of the wire format.** Non-Custom sections must appear in ascending id order — Type (1) → Function (3) → Export (7) → Code (10). Engines parse single-pass and rely on this.
- **Policy boundary:** which functions get exported is a *compiler* decision, not a *format* decision. Plex's current policy (mirrored from the WAT backend's [compiler.go:319-321](../compiler/compiler.go#L319-L321)) is "top-level functions yes, methods no."

Also fixed two carry-over bugs from lesson 26: `binary.LittleEndian.AppendUint64` → `PutUint64` (the former was silently zeroing every f64 constant because the returned slice was being discarded); and `signatureOf(m, m.Name)` → `signatureOf(m, n.Name)` so the method symbol name is now correctly `__StructName__methodName`.

End-to-end milestone: [end_to_end_test.go](../compiler/binary/end_to_end_test.go) parses plex source → `binary_wasm.CompileProgram` → `wazero.NewRuntime().Instantiate()` → `mod.ExportedFunction("forty_two").Call(ctx)` → returns 42.

**Implications:** The skeleton + Type + Function + Code + Export pipeline is complete. The user can now produce runnable WASM from plex, limited only by which AST nodes the `bodyEncoder.walk` switch handles. Lesson 28 expands that switch to cover params (via `*Identifier`) and binary arithmetic — enough to compile `fn add(a: i64, b: i64): i64 { return a + b; }`. After that, locals via `let` (lesson 29) require populating the code entry's locals vec for the first time.
