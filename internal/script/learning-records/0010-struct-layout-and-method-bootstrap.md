---
name: lesson-10-complete-method-bootstrap
description: Lesson 10 complete (struct layout + field access). Bootstrapped impl-block work — methods compile, name mangling in place, but two latent bugs remain.
metadata:
  type: project
---

Lesson 10 shipped. Working `structs.wat` shows correct layout for `struct Point { x, y }` —
allocate 16 bytes, store fields at offsets 0 and 8, leave base pointer on stack.

User then jumped straight into Lesson 11 territory (methods on structs) on their own:

- Added `*script.StructImplStatement` case in `Compile`. Sets `c.implBlock = n.Name`, walks the
  methods, clears `implBlock` at end.
- `FunctionDeclaration` checks `c.implBlock`: if non-empty, prepends `(param $self i32)` and uses
  the mangled name `__StructName__methodName`. Skips the `(export ...)` emit for methods.
- Added a `*script.MemberAccess` branch inside `FunctionCall`'s `Callee` switch — resolves the
  receiver's struct type via `typeofStruct`, checks the struct owns the method, compiles the
  receiver as the `self` arg, then `call $__StructName__methodName`.
- Switched `structs map[string]structDef` → `map[string]*structDef` because the impl block needs
  to mutate `Methods` after the struct is registered in the pre-pass. With value semantics, the
  append would have been lost.

Working end-to-end for the trivial case:

```
struct Point { x: float; y: float; }
impl Point { fn getX(): float { return self.x; } }
fn main() { let point = Point(1,1); point.getX(); }
```

→ produces `(func $__Point__getX (param $self i32) (result f64) ... )` and `call $__Point__getX`
with the receiver injected.

**Known bugs:**
1. `Methods` is appended twice per method — once in `FunctionDeclaration` (line ~148) and again
   in `StructImplStatement` (line ~559). Will surface as a false-positive duplicate detection if
   the duplicate-name check is tightened, or as memory waste / confusing debug output.
2. The duplicate-method-name check (`slices.Contains(d.Methods, n.Name)`) runs *before* the
   double-append, so it currently passes — but the moment we remove one of the appends, the
   check needs to stay in exactly one place.

**Next:** Lesson 11 will formalize the design (uniform calling convention, mangling rationale,
pointer-vs-value tradeoff) and use these bugs as the first exercise.
