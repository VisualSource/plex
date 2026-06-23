---
name: 0037-string-len-memarg
description: Lesson 37 — s.len() via i32.load + i64.extend_i32_s; method-call AST dispatch and memarg encoding
metadata:
  type: project
---

# String Length via i32.load — Binary Backend

User completed lesson 37. `TestEndToEnd_StringLen` passes for `"hello"`, `"wonderful"`, and `""`.

Concepts internalised:

- **Method-call AST shape.** `s.len()` parses as `FunctionCall{Callee: MemberAccess{Object: s, Field: "len"}, Args: []}`. The dispatch lives in the `FunctionCall` case (where the parentheses are), not in `MemberAccess` — because a bare `x.f` without parens means "value of a field", a different shape entirely. The binary backend's `FunctionCall` case now type-switches `n.Callee` into both `*script.Identifier` (direct call) and `*script.MemberAccess` (method/builtin dispatch).

- **The memarg encoding.** Every WASM memory instruction is followed by two ULEB128s: `align` (exponent — `2` means 2² = 4-byte alignment) and `offset` (constant added to the popped address). For loading a string's length prefix: `0x28 0x02 0x00` → `i32.load align=2 offset=0`. Align is a *hint* with an upper bound at the op's natural alignment; mismatched alignment is a perf concern, never a correctness one.

- **Sign-extension as type bridging.** `i32.load` pushes i32; the typechecker reports `len()` as `Int` (i64); the validator counts stack types and rejects the mismatch. `i64.extend_i32_s` (`0xAC`) widens with sign extension. For string lengths (always non-negative, always small), `extend_i32_s` and `extend_i32_u` give identical results. Match the WAT compiler's choice (`extend_i32_s`) for consistency.

## The collector-emitter invariant

Lesson 37 surfaced a *coverage bug* in lesson 36's string collector: `collectStrings`'s `FunctionCall` arm only walked `n.Args`, not `n.Callee`. `"hello".len()` has its only string literal inside the callee's `MemberAccess.Object`, so the collector missed it. At codegen time the `StringLiteral` case looked it up and got "string `hello` not in table".

**Invariant:** the AST traversal in `collectStrings` must dominate the traversal in `walk`. Any node `walk` recurses into during emission, `collectStrings` must recurse into during collection. The two passes are not independent — they share the same view of "what strings exist in this program".

**Future-proofing:** when arrays land, `ArrayLiteral.Elements` and `ArrayAccess.Target` will join the list of containers to walk. When structs land, `StructLiteral.Fields`. Each new container shape is a place the collector can silently fall behind the emitter and produce identical "not in table" errors. Refactoring both passes to share one visitor is the durable fix — but for now, the rule "update both when you add either" works.

## Implementation

- `OpI64ExtendI32S byte = 0xAC` added to opcodes.go.
- `*script.MemberAccess` arm in `FunctionCall` case: walk `callee.Object`, emit `OpI32Load`, align ULEB128 = 2, offset ULEB128 = 0, then `OpI64ExtendI32S`.
- `collectStrings` extended to walk `FunctionCall.Callee` *and* `MemberAccess.Object`.

**State after lesson 37:** strings work as both pointers (literal → i32 pointer) and as values (len → i64). `OpI32Load` is now genuinely used. Next gaps: comparison opcode completion (i32 lt/gt/le/ge, all f64 cmp); then heap + arrays — the gateway to non-static dynamic memory.
