---
name: 0049-arr-append-binary-backend
description: Lesson 49 — arr.append in binary backend; double-walk bug introduced
metadata:
  type: project
---

# arr.append in the Binary Backend

User completed lesson 49. `TestEndToEnd_ArrayAppend` passes (grow → 4, grow_read → 30, grow_twice → 6).

## What was implemented

`arr.append(elem)` uses `local.tee` to save the array pointer once, then computes the write address (`ptr + 4 + len*elemSize`), stores the new element, and increments the length stored at `ptr`. The `addSyntheticLocal` call must happen before any bytes referencing the local are emitted, because the local index is embedded in the `local.tee` operand.

## Double-walk bug introduced

The implementation walks `n.Args[0]` at lines 307–309 in `body.go` (before `addSyntheticLocal`) and then again at the correct position (line 335, between write_addr computation and store). This double-walk pushes the element value prematurely.

The tests pass because:
- The stray i64 from the first walk sits below the ptr i32 after the append body
- The Block-level `OpDrop` (for statement context) removes the top value (i32 ptr)
- The stray i64 then lives harmlessly below the `return` value, which WASM's polymorphic `return` ignores

The bug is masked by WASM's validation model, not correctness. Fix: delete lines 307–309. This will be addressed in lesson 50.
