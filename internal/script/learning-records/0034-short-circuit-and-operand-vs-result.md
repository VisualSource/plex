---
name: 0034-short-circuit-and-operand-vs-result
description: Lesson 34 — short-circuit AND/OR via if-as-expression, plus the operand-type-vs-result-type distinction surfaced by fixing the typechecker
metadata:
  type: project
---

# Short-Circuit && / || — Binary Backend

User completed lesson 34. `TestEndToEnd_ShortCircuitOr` and `TestEndToEnd_ShortCircuitAnd` pass. The full suite still passes.

Concepts internalised:

- **Short-circuit means lazy.** The bug was correctness-by-coincidence: `i32.or` on two bools (0 or 1) produces the right answer numerically, but the *RHS gets evaluated* regardless. With div-by-zero in the RHS, this is now observable as a trap that shouldn't fire. The fix is to special-case `&&`/`||` *before* walking the right side.

- **If-as-expression unlocks more than short-circuit.** A WASM `if` with block type `0x40` (empty) is a statement; with block type `0x7F` (i32 valtype) it pushes one i32 result. Both branches must produce exactly the declared result type — the validator enforces this statically. Short-circuit is just one application; ternary, expression-position `match`, etc. all reduce to the same shape.

- **Block types and value types share an encoding.** `0x7F` for i32, `0x7E` for i64, `0x7C` for f64 — the same bytes the type section uses, repurposed inline. The `types.go` constants (`ValI32`, etc.) are exactly the right values.

- **`blockDepth` discipline is invariant across `if` modes.** Whether `if` is empty or i32-result, it opens one label level. The helper bumps `blockDepth` before walking its branches so a hypothetical `break`/`continue` nested in a short-circuited RHS computes correct br depths. Lesson 31's discipline carries over without modification.

Implementation:
- `walkShortCircuit` helper on `bodyEncoder` — opens `OpIf, ValI32`, picks which side gets the constant, emits `OpEnd`.
- `*script.BinaryExpression` case branches to `walkShortCircuit` at the top when the operator is AND/OR.

## The deeper insight: operand type vs result type

Lesson 34's test exposed two latent bugs at once, and resolving them was the most pedagogically valuable part of the lesson.

**Typechecker bug** (since lesson 12): `*script.BinaryExpression`'s default arm returned `leftType` as the result type. This is correct for arithmetic (i64+i64 → i64) but wrong for comparisons (i64>i64 → bool). It never mattered because comparisons were only ever used directly as `if`/`while` conditions — never composed inside another typed expression. The first `||` over a comparison broke it. Fix: route all comparison operators (`==`, `!=`, `<`, `>`, `<=`, `>=`) through the same `bool`-returning arm as AND/OR.

**Codegen bug** (introduced by the typechecker fix): the binary backend dispatched opcodes using `n.GetType()` — the result type. After the typechecker fix, comparisons report `bool` as their result, and `cmpOpcode(Bool, >)` is nonsense. Fix: dispatch on the *operand* type via `n.Left.(script.Expression).GetType()`. Same one-line fix is needed in the WAT compiler at compiler.go:198 — currently dormant because wat2wasm is not on PATH.

**The principle:** a binary expression has two type questions, not one.

- *What goes in?* → operand type. Drives opcode selection. Use `n.Left.GetType()`.
- *What comes out?* → result type. Drives the parent's type-checking. Use `n.GetType()`.

Conflating them works only as long as every operator has `result = leftOperand`. The moment one operator breaks that invariant, the conflation collapses. Ask the AST the right question for the question being asked.

**State after lesson 34:** AND/OR are lazy; comparisons type correctly; opcode dispatch is honest about what type it's looking up. Next gaps: unary operators (no `*script.UnaryExpression` case in body.go yet), and the WAT backend has the same `n.GetType()` conflation waiting for someone to install `wat2wasm` and find it.
