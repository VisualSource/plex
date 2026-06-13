# Structs, arrays, break, and builtins all implemented and tested

User implemented StructType/StructValue/StructImplStatement, ArrayValue with append/remove/len as Go methods, breakSignal parallel to returnSignal, TernaryExpression, MemberAssignment, StringLiteral, and NewGlobalEnv with print/str. Three eval tests passing: multiplication via while loop, struct construction + method call, while/break.

One subtle issue: in the parser's parsePostfix, ArrayAccess.Object = the index expression and ArrayAccess.Field = the thing being indexed — names are swapped from what you'd expect. The eval handles it correctly but the naming is confusing; worth cleaning up.

**Implications:** Interpreter is feature-complete for the core language. User has a WASM compiler stub already. Ready for Lesson 6: introduction to code generation — the mindset shift from evaluating trees to emitting WebAssembly (WAT format, stack machine model, compile functions mirroring Eval).
