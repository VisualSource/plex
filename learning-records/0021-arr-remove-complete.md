---
name: arr-remove-complete
description: Lesson 21 complete — arr.remove(i) with internal shift loop, correct label stack handling
metadata:
  type: project
---

User implemented arr.remove(i) from Lesson 21.

Implemented:
- 5 temp locals per call: arrTmp (i32), idxTmp (i32), retTmp (elemType), lenTmp (i32), curTmp (i32)
- Step 1: save arr ptr via local.set, wrap i64 index to i32, load element at arr[i] into retTmp
- Step 2: compute new_len = old_len - 1, init cursor = idx
- Shift loop using `c.labelCount` directly (NOT pushBreakLabel) — labels $rem_break_N / $rem_loop_N
- Each iteration: push dst address (arr+4+cursor*elemSize), load src value (arr[cursor+1]), store
- cursor++ then br back to loop
- Step 3: store new_len back at arr[0]
- Step 4: local.get retTmp (return value on stack)

**Why:** Using c.labelCount directly (without pushBreakLabel) keeps the internal loop invisible to the user's break stack. A user-level break inside a while loop that calls arr.remove() correctly escapes the while loop, not the shift loop.

**How to apply:** The pattern of "compiler-generated loop with internal labels" applies any time a builtin method needs iteration. Same pattern would work for arr.insert(i, v) or string operations.
