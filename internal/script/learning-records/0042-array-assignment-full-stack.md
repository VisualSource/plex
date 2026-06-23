---
name: 0042-array-assignment-full-stack
description: Lesson 42 — arr[i] = x as a full-stack feature (parser + AST + typechecker + codegen)
metadata:
  type: project
---

# Array Element Assignment — Full-Stack Lesson

User completed lesson 42. `TestEndToEnd_ArrayAssignment` passes. `write_then_read`, `rotate`, and `sort_two` all work — the binary backend can now express in-place mutation, which means sorting algorithms are writable in plex.

Concepts internalised:

- **A language feature is a coordinated change across every stage that touches a program.** Today's `arr[i] = x` required:
  - **AST:** new `ArrayAssignment{Target *ArrayAccess, Value AstNode}` node, parallel to (but not unified with) `MemberAssignment`.
  - **Parser:** one new `case *ArrayAccess` in `parseAssignment`'s type-switch.
  - **Typechecker:** one new arm calling `checkExpression(n.Target)` for the side effect of populating the target's element type, then matching it against the value's type.
  - **Codegen:** one new case mirroring `ArrayAccess` point-for-point — same `wrap → mul → add` address arithmetic, same `+4`-via-memarg trick, terminal `store` instead of `load`.

- **Cross-stage discipline is the real skill.** Each stage's individual change was tiny. The discipline was making them consistent — same node shape used by every stage, same field semantics everywhere, no gaps between "what the parser produces" and "what codegen consumes".

- **The shape generalises.** Imports, modules, generics, async/await, pattern matching — every "add a language feature" task decomposes into "new AST node + parse rule + check rule + emit rule". Lesson 42 is one instance of a pattern that recurs throughout compiler engineering.

## Two design decisions worth pinning

### Two nodes, not one
`MemberAssignment` has a string `Field`; `ArrayAssignment` has a runtime-computed `Index`. Forcing one node to hold either would mean nullable fields and `if Field != "" else use Index` dispatch in every consumer. Two clean nodes beat one Frankenstein. **Heuristic**: when "field" semantics meaningfully differs from "the other field" semantics, the union node loses more than it gains.

### Don't implement `Expression` for statement-shaped nodes
Lesson 33's `Block` arm emits `OpDrop` after any statement that satisfies `script.Expression` with a non-void type. `ArrayAssignment` produces no stack value (the store consumes both operands and pushes nothing). If it accidentally implemented `GetType`/`SetType`, the drop logic would emit a stray `drop`, corrupting the stack. **The cleanest signal that "this is a statement" is not implementing the `Expression` interface.** Go's opt-in interface satisfaction makes this trivially safe — just don't add the methods.

### The typechecker's side-effectful walk
Calling `checkExpression(n.Target)` in the new arm wasn't redundant — its *side effect* is populating `n.Target.type_` with the element type. Codegen later reads this via `n.Target.GetType()` to pick the right store opcode. **Skipping the call to avoid "redundant work" would silently break codegen.** A class of bug worth pinning: type information that's only correct because a particular walk visited it.

## What plex source can now express

```
fn bubble_sort(): int {
    let arr = [3, 1, 4, 1, 5, 9, 2, 6];
    let n = arr.len();
    let i: int = 0;
    while i < n {
        let j: int = 0;
        while j < n - i - 1 {
            if arr[j] > arr[j+1] {
                let tmp = arr[j];
                arr[j] = arr[j+1];
                arr[j+1] = tmp;
            }
            j = j + 1;
        }
        i = i + 1;
    }
    return arr[0];  // smallest element
}
```

Bubble sort. From plex source. Compiled to a binary WASM module. Running in wazero. The toy language has crossed a threshold — programs of real algorithmic interest now compile and run.

**State after lesson 42:** the binary backend covers reading and writing arrays, plus everything from lessons 28-41 below them. The most-impactful remaining language gap is structs (a multi-stage shape similar to today). The most-impactful safety gap is bounds checking — the current `arr[i]` reads garbage for out-of-bounds indices instead of trapping. Either is a viable next lesson.
