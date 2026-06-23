---
name: 0030-if-statements-comparisons
description: Lesson 30 — comparison opcodes, structured if/else, and blocktype 0x40 in the binary WASM backend
metadata:
  type: project
---

# If Statements & Comparisons — Binary Backend

User completed lesson 30. The binary backend now handles `*IfStatement` and comparison `BinaryExpression`s, enabling `max(a, b)` to compile and run through wazero.

Concepts internalised:

- **Comparison opcodes always produce i32.** `i64.lt_s` takes two i64 operands but pushes a 1 or 0 as i32. `if` then consumes that i32. This asymmetry requires a separate `cmpOpcode` function rather than extending `arithOpcode`.

- **`cmpOpcode` takes the operand type, not the result type.** `i64.lt_s` (0x53) and `i32.lt_s` (0x48) are different binary instructions. The operand kind selects the right instruction family; the result is always i32 regardless.

- **WASM structured control flow uses nested blocks.** `if 0x40 ... else ... end` — no labels, no arbitrary jumps. The engine validates nesting in one pass, which is the basis for WASM's sandbox safety guarantee.

- **Blocktype 0x40 = empty result.** Immediately follows the `if` opcode. Signals that the if block leaves nothing on the stack. Statement-level ifs always use 0x40; expression-level ifs use a valtype (0x7E for i64, 0x7F for i32).

- **`walk` is recursive.** The `IfStatement` case emits the framing bytes (`if`, `else`, `end`); the body content is handled by the same `walk` switch via `n.Body` and `n.Else`.

- **`BooleanLiteral` maps to i32Const 1/0.** WASM has no bool type; booleans in plex are i32 at the binary level.

Completed tests: `max(42, 17) = 42` (then branch) and `max(17, 42) = 42` (else branch).

**Next:** while loops (lesson 31). WASM has no `while` opcode — loops are encoded as `block`+`loop`+`br_if`+`br`. This introduces label depth arithmetic that also makes `break` and `continue` work correctly even when nested inside `if` blocks.
