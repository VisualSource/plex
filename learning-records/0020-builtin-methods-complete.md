---
name: builtin-methods-complete
description: Lesson 20 complete — arr.len(), str.len(), arr.append() implemented; string layout changed to length-prefix
metadata:
  type: project
---

User completed all tasks from Lesson 20.

Implemented:
- `arr.len()`: `i32.load (ptr)` → `i64.extend_i32_s` (length already at offset 0 from array literal)
- String layout changed: offset now points to the length field (4 bytes), data starts at offset+4. Prepass: `offset: c.dataPtr`, `dataPtr += 4 + len(value)`. Emission: WAT `(data …)` starts with `\XX\XX\XX\XX` little-endian length bytes
- `str.len()`: `i32.load (ptr)` → `i64.extend_i32_s` (same pattern as arr.len — string pointer now points to length field, matching array layout)
- `arr.append(elem)`: reads old count, computes write addr (`ptr + 4 + count*elemSize`), stores elem, increments count. Uses temp local to hold array ptr (used 3 times). No bounds check.
- Type switch added in `FunctionCall → MemberAccess` path before struct lookup: `TypeKind_Array`, `TypeKind_String`, then `TypeKind_Struct`

**Note on string layout choice**: User chose to make string pointer point to the length field (same as array), NOT to the data. This differs from the lesson suggestion (lesson had ptr point to data, length at ptr-4). The user's choice means `str.len()` is `i32.load(ptr)` — identical to `arr.len()`. Host-side `print` must now skip 4 bytes or use the length to find the data. No null terminator in the emitted data segments.

**Why:** Keeping strings and arrays consistent (both have length at offset 0) simplifies the compiler and makes the pattern uniform. The tradeoff is that host-side callers must understand the new layout.

**How to apply:** User favors consistency over backward compatibility when there's a clear structural win.
