---
name: 0045-struct-methods-impl-binary-backend
description: Lesson 45 — struct method dispatch in the binary backend; mangled names, implicit self, call-site codegen
metadata:
  type: project
---

# Struct Methods — Binary Backend Dispatch

User completed lesson 45. `TestEndToEnd_StructMethods` passes — `p.distance_squared()` (no args) and `p.scale(10.0)` (one f64 arg) both produce correct results. Full suite green.

Concepts internalised:

- **Infrastructure was already in place.** `collectSignatures` in `helpers.go` already iterated `*script.StructImplStatement`, called `signatureOf` with `implName`, and emitted one `funcSig` per method with a mangled name `__Struct__method` and an implicit `ValI32` prepended to params. `newBodyEncoder` already wired the implicit `self` into `b.locals["self"] = 0`. The only missing piece was the call-site dispatch in `walk`.

- **Dispatch is one new branch in the `MemberAccess` callee case.** When `FunctionCall.Callee` is a `*script.MemberAccess` whose object resolves to `TypeKind_Struct`:
  1. Walk the receiver object → pushes i32 pointer
  2. Walk each arg → pushes arg values
  3. `OpCall` with the funcidx of `__Struct__method`

- **Name mangling replaces a vtable.** Plex's type system knows the concrete type at every call site (no subtyping, no generics). One flat `funcIndices` map holds both free functions and mangled method names. `OpCall` with a static funcidx is all that's needed — `call_indirect` would only be needed for runtime dispatch which plex doesn't have yet.

- **Self is an i32 local.** Inside a method body, `self` is index 0 (for methods) or just another named local. `MemberAccess(Object=self, Field=x)` and `MemberAssignment(Object=self, Field=x)` both go through the same field-offset machinery as lesson 44 — `self` is not special to the walker.

## Typechecker gap discovered post-lesson

The typechecker's `collectSignatures` case for `*script.StructImplStatement` was not pre-registering method params:
```go
// Before fix:
def.Methods[fn.Name] = newFuncInfo(c.scope)
// After fix:
methodInfo := newFuncInfo(c.scope)
for i, arg := range fn.Params {
    methodInfo.Args[arg.Name] = &orderedItem{Pos: i}
}
def.Methods[fn.Name] = methodInfo
```
Without this, `method.GetArgsInOrder()` returns `[]`, so `checkArgs` would accept any arity for struct method calls. Fix staged in `typecheck.go` after lesson 45 commit.

## State after lesson 45

Binary backend covers: strings, arrays, structs (fields + methods), all primitive operations, control flow, function calls. Method call sites dispatch via mangled name + OpCall. Next: void methods that mutate via `self.field = ...`, and methods that call other methods via `self.other()`.
