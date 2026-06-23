---
name: 0046-void-methods-mutation-self-calls
description: Lesson 46 — void method trap bug (OpUnreachable), self-mutation, self-calls, typechecker method-arg fix
metadata:
  type: project
---

# Void Methods, Mutation, and Self-Calls

User completed lesson 46. `TestEndToEnd_MutatingMethod` (33.0) and `TestEndToEnd_MethodCallsMethod` (150.0) pass. Full suite green (23 tests).

## Bug found and fixed: void method trap

`encodeCodeEntry` in `section.go` always emitted `OpUnreachable` before `OpEnd` as a WASM-validator safety net. For non-void functions this is harmless — after `return`, execution never reaches it. For void methods with no `return`, execution falls through to `OpUnreachable` → runtime trap.

Fix: `if len(sig.results) > 0 { append OpUnreachable }`. 3-line change.

## Why self-mutation and self-calls work for free

- `self.field = ...` inside a method: `MemberAssignment` walker from lesson 44 treats `self` like any struct-typed variable — it's just `locals[0]` (i32 pointer). No new codegen.
- `self.other()` dispatch: the `MemberAccess` callee branch already handles any expression as the receiver object. `Identifier("self")` resolves to `locals[0]`. Same mangled-name lookup. No new codegen.

## Typechecker fix (staged from lesson 45)

`collectSignatures` for `*script.StructImplStatement` wasn't pre-registering method params. `method.GetArgsInOrder()` returned `[]`, so `checkArgs` accepted any arity. Fixed by adding the param-position loop (matching the pattern already used for `FunctionDeclaration`).

## State after lesson 46

Binary backend covers: strings, arrays, structs (fields + methods, mutation, chaining), primitives, control flow, function calls. Next gap identified: arrays of structs — struct constructors nested inside array literals fail due to double-walking of args.
