---
name: 0053-string-concatenation
description: Lesson 53 — string + in binary backend; type-directed dispatch, memory.copy, 5 synthetic locals
metadata:
  type: project
---

# String Concatenation

Lesson 53 adds `string + string` to the binary backend.

## The gap

`arithOpcode` has no case for `TypeKind_String`. The current `BinaryExpression` handler walks both operands then tries to apply a single opcode — impossible for string concat, which needs to allocate and copy.

## Refactor: type before walk

The type check is moved before the walk calls so `walkStringConcat` can take full control of evaluation order. Both the `leftExpr`/`opType` extraction and the early exit happen before either operand is walked. The existing numeric/comparison path is unchanged; remove the now-duplicate extraction that was after the walk.

## walkStringConcat — five synthetic locals

| Local | Holds |
|---|---|
| leftPtr | i32 pointer to left string |
| rightPtr | i32 pointer to right string |
| leftLen | byte count of left |
| rightLen | byte count of right |
| resultPtr | base of new allocation |

**Phase 1** — walk left → local.set leftPtr; walk right → local.set rightPtr.  
**Phase 2** — i32.load from each pointer to get the lengths.  
**Phase 3** — global.get 0; local.tee resultPtr; store newLen (leftLen+rightLen) at resultPtr; bump __heap_ptr = resultPtr + 4 + leftLen + rightLen.  
**Phase 4** — two `memory.copy 0 0` calls: left bytes to resultPtr+4, right bytes to resultPtr+4+leftLen.  
**Return** — local.get resultPtr.

## memory.copy (WASM bulk memory, WASM 2.0)

Opcode: `0xFC 0x0A 0x00 0x00`. Stack: (dst i32)(src i32)(len i32) → (). The two trailing 0x00 bytes are the destination and source memory indices. wazero supports bulk memory by default; no feature flag needed.

## local.tee pattern

`local.tee` stores the top-of-stack value into a local without consuming it. Using it after `global.get 0` saves `resultPtr` while keeping the address on the stack as the target for the immediately following `i32.store` — saves one instruction vs. `local.set` + `local.get`.

## Tests

`TestEndToEnd_StringConcat` — `"hello" + " world"` → ptr to `"hello world"`.  
`TestEndToEnd_StringConcatVar` — string locals `a = "foo"`, `b = "bar"`, `a + b` → `"foobar"`.
