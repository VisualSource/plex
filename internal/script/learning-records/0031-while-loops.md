---
name: 0031-while-loops
description: Lesson 31 — block+loop+br two-level nesting, label depth arithmetic, break/continue in the binary WASM backend
metadata:
  type: project
---

# While Loops — Binary Backend

User completed lesson 31. The binary backend now handles `*WhileStatement`, `*BreakStatement`, and `*ContinueStatement`. `collectLocals` was extended to recurse into while and if bodies.

Concepts internalised:

- **WASM has no `while` opcode.** Loops are encoded as a `block`+`loop` two-level nesting. The `block` is the break target (its label points to its END); the `loop` is the continue target (its label points to its START). `br 0` inside a `loop` is a backward jump that restarts it.

- **The negate-then-branch pattern.** The while condition produces an i32 (0=false, 1=true). `i32.eqz` flips it, then `br_if 1` exits the outer block when the result is 1 (i.e., when the condition was false). Fall-through continues into the body.

- **Label depth is relative, not absolute.** Depth 0 = innermost enclosing block/loop/if. Each `block`, `loop`, or `if` instruction adds one level. `break` outside any extra nesting = `br 1`; inside one `if` = `br 2`. The `blockDepth` counter on `bodyEncoder` tracks open levels; `loopStack` records the depth at which each loop's labels were created so `br_if` and `br` depths can be computed at the point of any `break`/`continue`.

- **`IfStatement` must also track `blockDepth`.** The `if` instruction opens its own label level. Without `blockDepth++/--` in the `IfStatement` case, a `break` inside a conditional would emit the wrong depth and fail WASM validation.

- **`collectLocals` must recurse into while/if bodies.** The locals vec is emitted before any bytecode. A `let` declaration inside a loop that isn't pre-scanned produces an out-of-bounds `local.set` index — wazero rejects the module at validation, not at runtime.

Implementation:
- New fields on `bodyEncoder`: `blockDepth int`, `loopStack []loopEntry`
- `loopEntry{blockLabel, loopLabel}` stores the absolute depth boundaries for break/continue
- Depth formula: `br depth = uint32(b.blockDepth) - label`

Completed tests: `sum(10)=55` (10 iterations), `sum(0)=0` (zero iterations), `sum(1)=1` (one iteration).

**Next:** function calls (lesson 32). WASM `call funcidx` pops arguments from the stack and pushes the return value. Requires threading a `funcIndices map[string]uint32` into `bodyEncoder` so call expressions can resolve names to indices at emit time.
