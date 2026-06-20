---
name: drop-problem-complete
description: Lesson 16 complete — drop logic added to Block compiler, memory section emitted for heap, first plex programs run in WASM
metadata:
  type: project
---

User completed all tasks and went beyond:
- Added drop check to Block compiler (excludes VariableDeclaration and ReturnStatement)
- Extended memory section emission from `len(strings) > 0` → `needsHeap || len(strings) > 0`
- Validated output with `wat2wasm` and ran compiled WASM in Node.js
- `add(3.0, 4.0) = 7`, `double(5.0) = 10` — first successful end-to-end execution

**Why:** WASM requires a memory declaration before any memory access (store/load). Previously the heap could be used without a `(memory 1)` declaration, producing valid-looking but unrunnable WAT.

**How to apply:** Whenever you add a new use of linear memory, check whether the memory section declaration logic needs updating.
