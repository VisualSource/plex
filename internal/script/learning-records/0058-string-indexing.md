---
name: 0058-string-indexing
description: Lesson 58 — string indexing s[i]; two bugs fixed (typechecker return type, array vs byte addressing); walkStringIndex; load8_u with static offset
metadata:
  type: project
---

# String Indexing (`s[i]`)

Lesson 58 fixes two pre-existing bugs and adds `walkStringIndex` to the binary backend.

## Bug 1: Typechecker (typecheck.go)

`ArrayAccess` on a `TypeKind_String` identifier currently returns `TypeKind_String` as the element type. Fixed to return `TypeKind_Int` — the byte value (0–255) held as a plex integer.

## Bug 2: Binary backend (body.go)

After the typechecker fix, `s[i]` has element type `TypeKind_Int` (size 8). The existing `ArrayAccess` handler would multiply the index by 8 and use `i64.load` — reading 8 bytes at an 8× offset instead of 1 byte at a 1× offset.

Fix: at the top of the `ArrayAccess` case, check whether the target's type is `TypeKind_String` and dispatch to `walkStringIndex`.

## Memory layout comparison

```
int[]:  [len:i32] [elem0:i64] [elem1:i64] ...   element at base + 4 + i*8
string: [len:i32] [byte0] [byte1] [byte2] ...   byte at base + 4 + i
```

Both share the same bounds-check structure (compare index to i32 at offset 0), but differ in element addressing.

## walkStringIndex

1. Save string pointer to synthetic local `strBase`
2. Save index (i64 → i32 wrap) to synthetic local `strIdx`
3. Bounds check: `strIdx >= i32.load(strBase, offset=0)` → unreachable
4. Load: `i32.load8_u` at address `strBase + strIdx`, with `offset=4` baked into the instruction (skips the 4-byte length header without an extra add)
5. Extend to i64: `i64.extend_i32_s`

## Static memory offset

The `offset` immediate in WASM load/store instructions adds a constant to the stack address without consuming an extra stack slot. `i32.load8_u align=0 offset=4` is one instruction equivalent to `i32.const 4; i32.add; i32.load8_u align=0 offset=0` — saves one instruction and one stack slot.

This is the same offset trick used in `i32.load align=2 offset=4` for array element reads.

## Tests added

`TestEndToEnd_StringIndex`:
- `"hello"[0]` → 104 (`h`)
- `"hello"[4]` → 111 (`o`)
- `s[1]` on string variable → 101 (`e`)
- `s[0] + s[1]` on `"AB"` → 131 (65 + 66) — confirms int arithmetic on results
- Out-of-bounds: `s[100]` on short string → trap
