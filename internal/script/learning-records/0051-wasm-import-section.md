---
name: 0051-wasm-import-section
description: Lesson 51 — WASM import section; imported functions occupy lowest indices before local functions
metadata:
  type: project
---

# WASM Import Section

User completed lesson 51. `TestEndToEnd_Import_Print` passes — plex can call a host function (`plex:console.print`) from compiled WASM.

## What was implemented

Four coordinated changes:

1. **`importSig` struct** added to `helpers.go` — carries `modName`, `fieldName`, `funcName`, `params`, `results`.
2. **`collectImports`** scans the AST for `ImportStatement` nodes with source `"plex:console"` and returns `importSig` entries. A `string` argument maps to `ValI32` (pointer into linear memory).
3. **`encodeImportSection`** in `section.go` — emits section ID `0x02`, count, then for each import: encoded module name, encoded field name, kind byte `0x00` (function), and type index (the import's position in the unified type section).
4. **`module.go` restructured** — imports first in the type section, then local sigs. The `funcIndexes` map is built with imports at `[0 … numImports-1]` and locals at `[numImports … ]`. A fix-up loop rewrites export entries from local-relative indices to absolute indices from `funcIndexes`. `encodeFunctionSection` accepts a `typeOffset` argument so local function type indices are correct after the import types are prepended.

## Critical invariant

Imported functions **must** occupy the lowest function indices. Assign local indices only after the import count is known. If the compiler assigns local indices 0, 1, 2 and then adds an import at 0, every existing `call` instruction in the code section is wrong.

## Export fix-up

`collectSignatures` sets `idx` as a local-relative index (`uint32(len(funcs))`). The fix-up loop:

```go
for i := range exports {
    if exports[i].kind == ExportFunc {
        exports[i].idx = funcIndexes[exports[i].name]
    }
}
```

…converts these to absolute indices after `funcIndexes` is built. This also correctly handles the future case where function export indices shift because of added imports.

## String pointer convention

A plex `string` argument to an imported function is passed as `ValI32` — an offset into the module's linear memory. Layout: `[ptr+0..ptr+3]` = length (u32 LE), `[ptr+4 .. ptr+4+len]` = UTF-8 bytes. The host reads both out of `api.Module.Memory()`.
