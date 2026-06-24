---
name: 0055-break-continue-binary
description: Lesson 55 — break and continue in the binary backend; loopStack absolute-depth mechanism; modulo operator; end-to-end tests
metadata:
  type: project
---

# Break and Continue in the Binary Backend

Lesson 55 adds end-to-end tests for break and continue, which were already implemented in body.go but never verified at the WASM execution level.

## Status before this lesson

`BreakStatement` and `ContinueStatement` are handled in body.go (lines 231-246). The WAT compiler has `TestBreak` and `TestContinue`, but `end_to_end_test.go` has neither. Modulo `%` is in arithOpcode (i64.rem_s 0x81, i32.rem_s 0x6F) but also untested end-to-end.

## The WASM constraint

WASM has no goto. `br n` branches outward to the nth enclosing structured construct (0 = innermost). For a `loop`, br 0 jumps to the loop's beginning. For a `block`, br 0 exits past its `end`.

## While loop structure

```
block      ← break target  (blockLabel = absolute blockDepth here)
  loop     ← continue target (loopLabel = absolute blockDepth here)
    <condition>; i32.eqz; br_if (to exit block)
    <body>
    br 0   ← always loop back
  end
end
```

## The loopStack

Each WhileStatement pushes a `loopEntry{blockLabel, loopLabel}` onto `b.loopStack`. Both are stored as the absolute value of `b.blockDepth` at the time each construct was opened.

## Relative depth formula

At the point break/continue emits a branch:

- **break**: `b.blockDepth - entry.blockLabel`
- **continue**: `b.blockDepth - entry.loopLabel`

This formula is correct at any nesting depth. Each `if`/`block`/`loop` entered between the while construct and the break/continue site adds 1 to `b.blockDepth`, and the same 1 is subtracted when computing the relative depth.

## Example: break inside a nested if

If one `if` is open when break emits, blockDepth = loopLabel + 1. Formula: (loopLabel+1) - blockLabel = (loopLabel+1) - (loopLabel-1) = 2. In WASM: br 2 skips past the if (br 0 = if, br 1 = loop, br 2 = block exit). Correct.

## Tests added

`TestEndToEnd_Break`: array [10,20,30,40,50], find first >25 → 30. Also: no match → -1.  
`TestEndToEnd_Continue`: sum_evens(10) = 30 (2+4+6+8+10) using `i % 2 != 0` to skip odds.

The continue test exercises modulo as a side effect.

## blockDepth discipline

Every `block`, `loop`, `if` increments; every `end` decrements. Net change per construct = 0. The loopStack's absolute labels become stale if blockDepth is unbalanced — silent wrong results, not traps.
