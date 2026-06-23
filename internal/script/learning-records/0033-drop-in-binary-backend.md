---
name: 0033-drop-in-binary-backend
description: Lesson 33 — OpDrop and statement-position calls; recursion confirmed in the binary backend
metadata:
  type: project
---

# Drop in the Binary Backend

User completed lesson 33. Two tests now pass that didn't (or couldn't) before:

- `TestEndToEnd_Recursion` — `fib(10) = 55`, confirming that lesson 32's pre-built `funcIndices` map gives recursion for free.
- `TestEndToEnd_CallAsStatement` — `square(n);` as a discarded statement compiles to valid WASM, the stack balances, and `warmup(10) = 11`.

Concepts internalised:

- **Stack balance is the codegen invariant.** A WASM function body must leave exactly its declared results on the stack. The drop problem isn't about function calls specifically — it's about any expression whose result is unused. Calls just happen to be the first node that produces a value with no other consumer in plex's grammar.

- **The fix lives in the parent.** A `FunctionCall` cannot know whether its result will be consumed; only the surrounding context does. The `*script.Block` case in `walk` is the right place because the parent of every statement is a block. Same pattern as the WAT compiler in `compiler.go`.

- **The exclusion list matters.** `VariableDeclaration` and `AssignmentExpression` both emit `local.set` which already pops one value; an additional drop would pop the slot below it. `ReturnStatement` transfers its value to the caller. Anything else that satisfies `script.Expression` with a non-void type needs an explicit drop.

- **WASM has parametric instructions.** `drop` (0x1A) and `select` (0x1B) operate on the stack without caring about value types — they're encoded once and work for any of i32/i64/f32/f64.

Implementation:
- `OpDrop byte = 0x1A` added to opcodes.go
- The `*script.Block` case in body.go gained a post-walk check: if `s` satisfies `script.Expression`, has a non-void type, and isn't a Variable/Assignment/Return, emit `OpDrop`.

**State after lesson 33:** the binary backend's statement-level codegen is now stack-balanced for every expression node currently handled. Calls, recursion, and direct calls all work. Next gap: unary operators (`!`, unary minus) and short-circuit evaluation for `&&`/`||` — both currently bitwise and eager.
