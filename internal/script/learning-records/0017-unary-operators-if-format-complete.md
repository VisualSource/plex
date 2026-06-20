---
name: unary-operators-if-format-complete
description: Lesson 17 complete — UnaryExpression compiler, WAT (if fix, count program running in WASM
metadata:
  type: project
---

User completed all tasks from Lesson 17:
- Fixed `(if` format (was emitting bare `if`, mixed linear and folded syntax)
- Implemented `UnaryExpression` in compiler: prefix `-` (f64.neg / 0-x pattern for i64), prefix `+` (no-op), postfix `++` (double-push pattern), postfix `--`
- Compiled and ran `count(10) = 10` end-to-end in Node.js
- Updated compiler snapshots

**Why:** The WAT `if` instruction has two text formats — linear (closes with `end`) and folded (uses parentheses). Mixing them causes `wat2wasm` to reject the output. Integer negation has no direct WAT instruction, requires synthesising `0 - x`.

**How to apply:** User is comfortable with double-push patterns and WAT stack mechanics. Can build on this for future operator work.
