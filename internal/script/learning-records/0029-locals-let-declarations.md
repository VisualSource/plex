---
name: 0029-locals-let-declarations
description: Lesson 29 — locals vec, two-phase pre-scan, and local.set in the binary WASM backend
metadata:
  type: project
---

# Locals via `let` — Binary Backend

User completed lesson 29. The binary backend now handles `*VariableDeclaration` and `*AssignmentExpression`, and `encodeCodeEntry` emits a real locals vec instead of a hardcoded `0`.

Concepts internalised:

- **The code entry format demands locals before bytecode.** `code = vec(local_group) expr end` — all locals must be declared at the top. There is no inline declare instruction. This forces a two-phase approach even though the AST is one tree.

- **Two-phase: collect then emit.** `collectLocals` (Pass 1) walks the AST to assign indices and record valtypes. The `walk` switch (Pass 2) emits bytecode using those pre-assigned indices. This pattern — "output format has constraints the AST doesn't share, so pre-scan first" — recurs throughout compilers (JS hoisting, C forward decls, etc.).

- **Params and locals share one flat index space.** `len(b.locals)` at the time a `VariableDeclaration` is first encountered is always the next available index, since params are pre-populated in `newBodyEncoder`.

- **local.set (0x21) pops the stack.** `walk(init)` pushes the value; `local.set` consumes it. Net stack effect: 0 — correct for a statement.

- **`OpI64Mul` bug fixed** (was `0x7F`, should be `0x7E`). `OpI64Mul` and `OpI64DivS` had collided — any i64 multiplication was silently emitting the division opcode.

- **`OpLocalTee` noted** (0x22) — pushes AND sets, useful for `x = x + 1` inside an expression, but not needed yet.

Completed test: `accumulate(20, 21) = (20+21)+1 = 42`, exercising two chained locals with distinct indices.

**Next:** comparison operators + if statements (lesson 30). Comparisons extend `BinaryExpression` handling with a `cmpOpcode` function — they always produce i32 regardless of operand type. If statements introduce WASM's structured control flow: `if 0x40 ... else ... end` with blocktype `0x40` (no result) for statement-level ifs.
