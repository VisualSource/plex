---
name: binary-operators-bool-complete
description: Lesson 18 complete — bool type, comparison operators, logical AND/OR, test helpers
metadata:
  type: project
---

User completed all tasks from Lesson 18:
- Added `bool` type to typesystem and tokenizer (`true`/`false` keywords)
- Implemented `BooleanLiteral` in compiler (i32.const 1 / i32.const 0)
- Implemented comparison operators: `<`, `>`, `<=`, `>=`, `==`, `!=` with signed/unsigned variants per type
- Implemented logical `&&` (i32.and) and `||` (i32.or)
- Added `helpers_test.go` with compiler helper unit tests
- Snapshots updated; bool_test.wat verified

**Why:** Booleans in WAT are represented as i32 (0=false, 1=true). Comparison operators must emit signed vs unsigned variants (e.g. `i64.lt_s` vs `i64.lt_u`) based on the operand type. Logical AND/OR on booleans is just bitwise i32 and/or since the values are always 0 or 1.

**How to apply:** User is now comfortable with type-directed instruction selection. The pattern of `prefix := getWasmType(t); emit(prefix + ".add")` is well understood. Can build on this for operator work and type coercions.
