---
name: break-ternary-complete
description: Lesson 19 complete — BreakStatement with named label stack, TernaryExpression with value-producing if
metadata:
  type: project
---

User completed all tasks from Lesson 19 ahead of the lesson being written — implementation was already in the codebase when the lesson was delivered.

Implemented:
- `labelCount int` and `breakStack []string` on `Compiler` struct
- `pushBreakLabel() (string, string)` — allocates `$blockN`/`$loopN` pair, pushes break label
- `popBreakLabel()` — pops and decrements counter (counter reuse is safe since loops don't overlap)
- `currentBreakLabel() string` — peeks top of stack
- `WhileStatement` updated to use named labels; `defer popBreakLabel()` for cleanup on error
- `BreakStatement` case: guards against `len(breakStack) == 0`, emits `br $blockN`
- `TernaryExpression` case: compile condition → `(if (result TYPE) (then …) (else …))`

**Why:** Relative branch indices (`br 0`, `br_if 1`) don't work when `break` is nested inside an `if` — the depth changes. Named labels let `break` target the outer block regardless of nesting depth. Value-producing WAT `if` requires `(result TYPE)` annotation; the typechecker already enforces both branches have matching types.

**How to apply:** User is comfortable with the label stack pattern and with WAT's folded control flow syntax. The same stack pattern could extend `continue` (would need a separate `loopStack` for `$loopN` labels).
