---
name: 0054-string-equality
description: Lesson 54 — string == and != in binary backend; i32.load8_u, block+loop byte comparison, i32.eqz for !=
metadata:
  type: project
---

# String Equality

Lesson 54 adds `string ==` and `string !=` to the binary backend.

## The gap

`cmpOpcode` has no case for `TypeKind_String`. Both `==` and `!=` on strings route to a new `walkStringEq` method via the string early-exit block in `BinaryExpression`.

## New opcode

`OpI32Load8U = 0x2D` — loads one byte from memory and zero-extends to i32. Used to read individual string bytes. `align=0, offset=0` (no alignment requirement for byte loads).

## Control flow: block + loop + nested if

Six synthetic locals: leftPtr, rightPtr, leftLen, rightLen, cursor, isEqual.

Structure:
1. Set `isEqual = 1` (assume equal)
2. Read both lengths
3. Set `cursor = 0`
4. Enter outer `block` (b.blockDepth++)
5. **Length fast path**: `leftLen != rightLen` → set isEqual=0, `br 1` (exit block)
6. Enter `loop` (b.blockDepth++)
7. `cursor >= leftLen` → `br_if 1` (exit block, done, isEqual=1)
8. Load left byte and right byte via `i32.load8_u`
9. `i32.ne` → enter `if` (b.blockDepth++)
10. Set `isEqual = 0`; `br 2` (exit block from inside if — skips if+loop, exits block)
11. Exit `if` (b.blockDepth--)
12. `cursor++`; `br 0` (continue loop)
13. Exit `loop` (b.blockDepth--)
14. Exit `block` (b.blockDepth--)
15. `local.get isEqual`
16. For `!=`: append `i32.eqz`

## Branch depths (hardcoded, not from b.blockDepth)

From inside the loop body (block at D+1, loop at D+2):
- `br_if 1` → exits outer block (done)
- `br 0` → continues loop

From inside the nested if (block at D+1, loop at D+2, if at D+3):
- `br 2` → exits outer block

From inside the length-check if (block at D+1, if at D+2):
- `br 1` → exits outer block

These are relative to the internal structure and are hardcoded, not computed from b.blockDepth.

## blockDepth discipline

Must increment at each `block`, `loop`, `if` and decrement at each `end`. Net zero change. User-visible while loops store break/continue targets as absolute depth values; unbalanced depth corrupts those calculations when this expression appears inside a loop.

## != via i32.eqz

`isEqual` is 1 (equal) or 0 (not equal). `i32.eqz` returns 1 when its operand is 0 — precisely the inverted boolean needed for `!=`.

## Tests

`TestEndToEnd_StringEq`: same content (expect 1), same length different content (expect 0), different length (fast path, expect 0).  
`TestEndToEnd_StringNe`: inverted results.

The fast path for different lengths is important to test because it exercises the `br 1` from the length-check if — a different code path from the byte loop.
