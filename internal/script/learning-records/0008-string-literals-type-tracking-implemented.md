---
name: lesson-08-complete
description: Lesson 8 complete — string literals, two-pass collect, type-aware locals, resolveType all implemented
metadata:
  type: project
---

User went significantly beyond the lesson scope. Implemented:
- `stringEntry` + `collectStrings` two-pass approach for static string data sections
- `*script.StringLiteral` → `i32.const <offset>` in compiler
- `Local{Name, Type}` struct replacing plain `[]string` for locals
- `resolveType(AstNode) string` mapping plex types to WASM types (int→i64, float→f64, string/unknown→i32)
- Type-aware `collectLocals`: explicit type from annotation, implicit type from init expression (StringLiteral→i32, default→i32 for pointers)
- Type-aware params and return type in FunctionDeclaration

**Why notable:** User independently identified the type-mismatch problem (i32 addresses stored in f64 locals) and solved it before being taught.

**Pending issues:**
- BinaryExpression still always emits f64 operators — needs type-aware operator selection now that we have typed operands (tracked as TODO in code)
- NumberLiteral implicit type doesn't yet distinguish i64 vs f64 (another TODO)
- `collectStrings` doesn't deduplicate identical string literals (minor: same string gets two data section entries)
