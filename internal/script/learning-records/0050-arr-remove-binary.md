---
name: 0050-arr-remove-binary
description: Lesson 50 — arr.remove in binary backend; double-walk fix and internal shift loop
metadata:
  type: project
---

# arr.remove in the Binary Backend

User completed lesson 50. All three `TestEndToEnd_ArrayRemove` sub-cases pass (remove_middle → 20, remove_shifts → 20, remove_len → 2). The double-walk bug in `append` was fixed.

## Double-walk bug fixed

Deleted the premature `walk(n.Args[0])` call that appeared before `addSyntheticLocal` in the `"append"` case. The bug was masked because WASM's polymorphic `return` discards stack values below the top N return slots — the stray i64 was invisible to the validator.

## arr.remove implementation

Five synthetic locals: `arrLocal` (i32), `idxLocal` (i32), `retLocal` (element valtype), `lenLocal` (i32), `curLocal` (i32). The element to return is saved **before** the shift loop, because the first loop iteration overwrites `arr[idx]`.

## Internal shift loop — blockDepth discipline

The shift loop is a `block + loop` pair emitted in binary bytes. It is **not** added to `loopStack` (user `break`/`continue` must not reach it), but `b.blockDepth` **must still be incremented and decremented correctly**. User-visible while loops record their break/continue targets as absolute blockDepth values; an unbalanced internal depth would corrupt those branch calculations.

Branch depth arithmetic inside the loop (starting at depth D):
- emit OpBlock → D+1 → blockMark = D+1
- emit OpLoop  → D+2 → loopMark  = D+2
- br_if exit: D+2 − blockMark = 1
- br continue: D+2 − loopMark  = 0
- emit OpEnd × 2 → back to D
