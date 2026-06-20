---
name: type-error-diagnostics-complete
description: Lesson 14 complete — %T→%s fixed, Type.String() implemented, cascading nil-type errors suppressed
metadata:
  type: project
---

User completed all four Lesson 14 tasks:
- Changed `%T` to `%s` in `ReturnStatement` error messages in `typecheck.go`
- The remaining `%T` usages (`unhandled node %T`) are intentionally correct — they print the Go AST node type name
- All typechecker tests pass

**Why:** `%T` prints the Go *type name* of the variable, not its value — so every `TypeKind` printed as `"script.TypeKind"`. `%s` invokes the `String()` method and gives the actual kind name.

**How to apply:** In Go error messages, use `%T` only when you want the variable's type name for debugging/introspection. Use `%s` or `%v` to print the value.
