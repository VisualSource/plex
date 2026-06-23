---
name: 0047-arrays-of-structs
description: Lesson 47 — arrays of structs, double-walk bug fix, and method calls on array elements
metadata:
  type: project
---

# Arrays of Structs

User completed lesson 47. `TestEndToEnd_ArrayOfStructs` (30.0) and `TestEndToEnd_ArrayOfStructsMethods` (194.0) pass.

## The double-walk bug

Struct constructors nested inside array literals previously caused the constructor args to be walked twice — once by the array literal's element loop and again by `walkStructConstructor`. Fix: detect a `*script.FunctionCall` whose callee is a struct name inside `walkArrayLiteral` and call `walkStructConstructor` directly instead of recursing into `walk`.

## Why indexing a struct array works

`arr[i]` on a `Vec2[]` returns an i32 pointer (same as a standalone `Vec2` local). The element type stored in the array is always i32 (a pointer), so `I32Load` retrieves it. The resulting pointer is then used exactly like any other struct pointer — field loads and method dispatch are unchanged.

## State after lesson 47

Binary backend covers: primitives, control flow, functions, strings, arrays (primitive and struct), structs (fields + methods, mutation, self-calls), arrays of structs. Remaining gaps: ternary expressions, type cast expressions.
