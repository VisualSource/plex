# Typechecker scope chain implemented — push/pop pattern missing

User implemented the `Scope` struct with parent-pointer chain and `FuncInfo.Scope` correctly.
The data structure is right. The missing piece is the push/pop pattern in `checkStmt`:
the `FunctionDeclaration` case populates `fn.Scope.vars` with params but never sets
`c.scope = fn.Scope` before calling `checkStmt(n.Body)`, so identifier lookups inside
function bodies fail with "unknown identifier" on every parameter.

**Implies:** Lesson 13 should focus purely on the push/pop idiom — the fix is three lines.
Block-level scoping (if/while bodies) and return-type validation are natural stretch tasks
from the same lesson.
