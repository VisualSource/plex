---
name: 0052-export-keyword
description: Lesson 52 — export keyword; four-layer change from tokenizer to binary backend controlling WASM module visibility
metadata:
  type: project
---

# The export Keyword

Lesson 52 teaches the `export` keyword — controlling which plex functions are visible to the WASM host.

## Four-layer change

| Layer | File | Change |
|---|---|---|
| Tokenizer | `tokenizer.go` line 425 | add `"export"` to keyword case |
| AST | `ast_nodes.go` | add `Export bool` to `FunctionDeclaration` |
| Parser | `ast.go` `parseStatement` | add `export` case before `fn` case; consume `export`, call `parseFnDecl()`, set `decl.Export = true` |
| Binary backend | `helpers.go` `collectSignatures` | gate `exports = append(...)` on `n.Export == true` |

## Key insight: tokenizer gate

`export` must be in the tokenizer keyword list before the parser can test `isKeyword(t, "export")`. Without this, the tokenizer emits `TokenType_Ident`, `isKeyword` returns false, and the parser falls through to the default `exprStmt` case — treating `export fn foo()` as a broken expression statement.

## Unexported functions still compile

Removing an export entry does not remove the function from the code section. The function's body, local variables, and internal call graph are unchanged. Other plex functions in the same module call it via its absolute function index (module-internal, independent of the export table).

## Test suite migration

After gating exports on `n.Export`, all existing test programs that use bare `fn` produce no exports — every `mod.ExportedFunction("name")` call returns nil and panics. Migration: `sed -i 's/\bfn /export fn /g' end_to_end_test.go`. The word `fn` does not appear in Go code (Go uses `func`), so the replacement is safe inside the backtick string literals.

## idx fix-up still works

The new export entries intentionally omit the `idx` field (zero by default). The existing fix-up loop in `module.go` sets `exports[i].idx = funcIndexes[exports[i].name]` for every `ExportFunc` entry after `funcIndexes` is built. This is the same mechanism introduced in lesson 51 for the import-offset shift.

## Visibility test

`TestExport_VisibilityBoundary` verifies the invariant directly: a program with one `export fn` and one bare `fn` compiles; `ExportedFunction("public_fn")` returns non-nil, `ExportedFunction("private_fn")` returns nil.
