---
name: 0032-function-calls-binary-backend
description: Lesson 32 — call funcidx in the binary backend, funcIndices map threaded into bodyEncoder
metadata:
  type: project
---

# Function Calls — Binary Backend

User completed lesson 32. The binary backend now handles `*script.FunctionCall` for direct calls. `TestEndToEnd_FunctionCall` passes: `sum_squares(3, 4) = 25`, `sum_squares(5, 12) = 169`, `square(7) = 49`.

Concepts internalised:

- **The stack is the calling convention.** Walk args left-to-right (each pushes onto the value stack), then emit `call funcidx` (0x10). The engine pops N args, runs the callee, pushes the return value. No register conventions, no caller/callee save split — it's all stack.

- **Names → indices is a compile-time mapping.** WASM has no symbol table for internal calls. The funcidx is the function's 0-based position in the function section. A `funcIndices map[string]uint32` is built once from `sigs` (the slice already returned by `collectSignatures`) and shared read-only with every `bodyEncoder`.

- **Forward references work for free.** The map is built from the complete `sigs` slice *before* any body is encoded, so a function can call another that appears later in source. The validator checks indices at module load, not at parse time. By the same mechanism, **recursion already works** — a function finds its own name in `funcIndices` when walking its self-call. No fib test was written, but the design guarantees it.

Implementation:
- `OpCall byte = 0x10` added to opcodes.go
- `bodyEncoder` gained a `funcIndices map[string]uint32` field
- `newBodyEncoder`, `encodeCodeEntry`, `encodeCodeSection` all take the map as a parameter
- `*script.FunctionCall` case in `walk`: walks each arg in order, type-switches the callee (only `*script.Identifier` supported for now), emits `OpCall` + ULEB128 funcidx

**Latent stack-balance bug:** a call in *statement position* (e.g. `add(1, 2);` as a top-level statement) leaves the return value on the stack with no consumer. WASM validation rejects such modules. The current test only exercises calls in expression position (inside `return ... + ...`), so the bug is dormant. This is exactly the drop problem already learned for the WAT backend in lesson 16 — about to recur in lesson 33.

**State after lesson 32:** the binary backend now compiles params, locals, arithmetic, comparisons, if/else, while, break/continue, and function calls — enough for any computation expressible without strings/arrays/structs. The next missing piece is `drop` for statement-position calls; after that, multi-type plumbing (i32/bool/f64) and then linear memory for strings.
