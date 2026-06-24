---
name: 0057-increment-decrement
description: Lesson 57 — ++ and -- in binary backend; lvalue restriction; local.tee read-modify-write; pre-increment vs postfix semantics
metadata:
  type: project
---

# Increment and Decrement (`++` / `--`)

Lesson 57 adds `x++` and `x--` to the binary backend. The parser already produces `UnaryExpression{Operator: TokenType_Incrment/Decrement, Postfix: true}`. No tokenizer, parser, or typechecker changes needed.

## The lvalue concept

`++` and `--` are read-modify-write operators. They require an lvalue — a named storage location — not just a value. `(x+1)++` is rejected at codegen time because there is no local index to write back to. The restriction is enforced by a type assertion `n.Operand.(*script.Identifier)` in body.go.

A production compiler would catch this earlier (in a dedicated lvalue-checking pass or in the typechecker), giving better error messages. We defer it to codegen for simplicity.

## local.tee

`local.tee n` (OpLocalTee = 0x22): writes top-of-stack to local n without consuming it. Stack in: [value]. Stack out: [value]. Local n = value.

Used here: `local.get x; i64.const 1; i64.add; local.tee x` leaves the new value (x+1) on the stack while also storing it in x.

## Semantics: pre-increment, not C-style postfix

The implementation returns the NEW value (x+1), not the original. For statement use (the drop fires), this is invisible. For expression use (less common), the value differs from C's postfix `x++`.

True C postfix would require: `local.get x` (read), `addSyntheticLocal(ValI64)` (save original to tmp), `i64.const 1; i64.add; local.set x` (compute and store), `local.get tmp` (push original). This costs an extra synthetic local.

## Types supported

i64/int and i32. f64 excluded — floating-point increment by 1.0 is unusual and would need OpF64Const (8 bytes) rather than the compact 1-byte i64.const 1.

## No typechecker change

The typechecker's UnaryExpression handler propagates the operand type unchanged. For `x++`, the result type is the type of x — which is exactly what propagation gives. No special case needed.

## Tests added

`TestEndToEnd_Increment`: count_up (three i++; → 3), count_in_loop (while i < n { i++; } → n).  
`TestEndToEnd_Decrement`: count_down (while i > 0 { i--; } from n → 0).
