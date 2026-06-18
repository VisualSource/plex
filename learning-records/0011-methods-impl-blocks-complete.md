---
name: lesson-11-complete
description: Lesson 11 complete — methods, impl blocks, and self. Double-append bug fixed; at least one stretch task done (methods with params or method-calling-method).
metadata:
  type: project
---

User completed:

- **Task A (the fix)**: removed the redundant `Methods` append. Single source of truth for
  method registration now lives in one place — either in `FunctionDeclaration` (eager,
  per-method) or in `StructImplStatement` (lifted to a pre-loop over the impl block). Whichever
  branch they chose, `def.Methods` no longer has duplicates.
- **At least one of Task B or C**: either methods with struct parameters
  (`fn distanceTo(other: Point): float`) or a method calling another method on the same struct
  (`fn doubleX() { return self.getX() * 2; }`).

Conceptually internalized:

- Methods are sugar for functions with a hidden first arg and a mangled name. There is no
  separate dispatch machinery — just a regular `call` with one extra parameter.
- Name mangling (`__StructName__methodName`) is necessary because WASM has a flat function
  namespace; two structs with `area()` methods would collide otherwise.
- `structs map[string]*structDef` (pointer) is required because Go's map values aren't
  addressable — mutations through `map[k]` copies are silently lost. This came up because the
  impl block needs to mutate `Methods` during compilation.
- `self` is just a WASM local (slot 0), so `self.x` works automatically via Lesson 10's
  `MemberAccess` branch. This is the payoff of a uniform calling convention.

**Latent question raised by Task C (if done)**: if `doubleX` calls `getX` but `getX` is declared
*after* `doubleX` in the impl block, the eager registration breaks. The fix is a two-pass
compilation of the impl block (collect method names first, then compile bodies). This is the
same pattern as forward references for free functions.

**Next:** Lesson 12 — type checking as a separate pass. Pull `typeOf`, `typeofStruct`, and the
ad-hoc type mismatch checks out of codegen into a dedicated phase.
