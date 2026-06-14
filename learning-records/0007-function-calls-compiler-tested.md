---
name: lesson-07-complete
description: Lesson 7 complete — FunctionCall compiled, compiler test written and passing
metadata:
  type: project
---

User implemented *script.FunctionCall in the compiler (push args, call $name) and wrote compiler_test.go asserting WAT output contains expected instructions. All compiler tests pass.

**Why:** Closing the loop so functions can call each other; testing WAT output by string containment rather than value execution.

**How to apply:** Compiler now handles the full numeric subset of plex. Next lesson is linear memory for strings/arrays/structs.
