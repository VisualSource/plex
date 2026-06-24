---
name: 0059-power-operator
description: Lesson 59 — ** operator in binary backend; inline accumulator loop; right-associativity already handled by parser
metadata:
  type: project
---

# Power Operator (`**`)

Lesson 59 adds `**` to the binary backend. The tokenizer, parser, and typechecker already supported it — only codegen was missing.

## What Already Existed

- **Tokenizer**: `**` produces `TokenType_Power`.
- **Parser**: `parsePower()` handles it right-associatively: `a ** b ** c` → `a ** (b ** c)`.
- **Typechecker**: `BinaryExpression` default branch propagates the left type — correct since `int ** int → int`.

## Implementation: `walkPower`

WASM has no native integer power instruction. The implementation uses an accumulator loop — the same pattern you'd write by hand:

```
result = 1
while exp > 0 {
    result *= base
    exp--
}
```

Emitted as a `block`+`loop` pair using the same depth-relative label math as `WhileStatement`:
- `br_if` to exit the block when `exp <= 0`  (`i64.le_s`)
- `br 0` to jump back to loop top

Three synthetic locals: `base` (i64), `exp` (i64), `result` (i64). All allocated via `addSyntheticLocal(ValI64)` before the loop so they appear in the function's local section.

## Dispatch

In the `BinaryExpression` case, `TokenType_Power` is intercepted before the type extraction and `arithOpcode` path — similar to `&&`/`||` dispatching to `walkShortCircuit`:

```go
if n.Operator == script.TokenType_Power {
    return b.walkPower(n)
}
```

## Right-Associativity Confirmation

`2 ** 3 ** 2` parses as `2 ** (3 ** 2)` = `2 ** 9` = 512 — tested in `TestEndToEnd_Power/right_assoc`. The parser handles this correctly without any codegen change.

## Edge Case: Zero Exponent

`x ** 0` → the loop exits immediately (condition is already `0 <= 0`), leaving `result = 1`. Correct by construction.

## Tests Added

`TestEndToEnd_Power`:
- `square(4)` → 16
- `cube(3)` → 27
- `pow_zero(999)` → 1
- `right_assoc()` → 512
