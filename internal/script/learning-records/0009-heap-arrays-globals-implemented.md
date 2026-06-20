---
name: lesson-09-complete
description: Lesson 9 complete — arrays, heap, globals, import system, embedded helper plex files all implemented
metadata:
  type: project
---

User went significantly beyond the lesson scope. Implemented:
- `hasHeapObjects` pre-pass to emit heap infrastructure only when needed
- 8-byte-aligned heap start calculated from `c.dataPtr`
- `globalVars []string` so Identifier/AssignmentExpression emit global.get/set for globals
- `tempArrayVars []string` stack for auto-naming temp locals per ArrayLiteral
- Full `ArrayLiteral` compilation: alloc + length header + per-element stores with correct offsets
- Full `ArrayAccess` compilation: base + 4 + index*8, i32.wrap_i64, f64.load
- Type-aware `typeOf()` returning errors on mismatches
- `importModules()` for a plex import system with `plex:globals` and `plex:console` namespaces
- `loadHelper()` using `//go:embed` to load plex helper files compiled inline
- `helpers/global.plex` — the bump allocator written in plex itself, bootstrapped by the compiler

**Known bug to address:** `clear(c.tempArrayVars)` in FunctionDeclaration doesn't reset the slice length — should be `c.tempArrayVars = c.tempArrayVars[:0]`. Will surface when multiple functions use arrays.

**Next:** Lesson 10 — struct layout and field access (StructStatement compilation, struct registry, MemberAccess/MemberAssignment).
