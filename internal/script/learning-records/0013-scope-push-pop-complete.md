---
name: scope-push-pop-complete
description: Lesson 13 complete — scope chain push/pop pattern implemented in checkFunction and Block
metadata:
  type: project
---

`checkFunction` now correctly does: `prev := c.scope` → `c.scope = info.Scope` → `c.checkStmt(def.Body)` → `c.scope = prev`.
Block-level scoping (if/while bodies) was also added via `newScope(c.scope)` in the Block case of `checkStmt`.

**Why:** Without setting `c.scope = info.Scope` before checking the body, parameter lookups inside function bodies always returned "unknown identifier" because the checker was using the outer (global) scope.

**How to apply:** The push/pop idiom is the canonical way to handle nested scopes. Any new scope boundary (lambda, block expression, catch clause, etc.) should follow the same prev/push/check/pop pattern.
