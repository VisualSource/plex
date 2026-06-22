# Compiler Construction Resources

## Knowledge

- [Book: _Crafting Interpreters_ — Robert Nystrom](https://craftinginterpreters.com/)
  The gold standard hands-on compiler book. Free online. Use for: every stage from scanning to bytecode VM. Jlox (Java tree-walking) maps directly to our approach.

- [Book: _Writing An Interpreter In Go_ — Thorsten Ball](https://interpreterbook.com/)
  Same progression as Crafting Interpreters but in Go — directly applicable to this repo. Use for: Go-idiomatic parser and evaluator patterns.

- [Article: Pratt Parsing — Matklad](https://matklad.github.io/2020/04/13/simple-but-powerful-pratt-parsing.html)
  The clearest explanation of Pratt (top-down operator precedence) parsing. Use for: expression parsing with correct precedence without deeply nested grammar rules.

- [Wikipedia: Recursive Descent Parsing](https://en.wikipedia.org/wiki/Recursive_descent_parser)
  Quick reference for the formal definition. Use for: understanding the connection between BNF grammar rules and parse functions.

- [WebAssembly Core Spec — §5 Binary Format](https://webassembly.github.io/spec/core/binary/index.html)
  The normative reference for the `.wasm` binary layout. Use for: every byte you emit when writing a binary backend — section IDs, integer encodings, value/type encodings.

- [WebAssembly Core Spec — §5.2.2 Integers (LEB128)](https://webassembly.github.io/spec/core/binary/values.html#integers)
  The formal definition of how unsigned/signed LEB128 is used in WASM. Use for: implementing the LEB128 helpers that underpin every section.

- [Wikipedia: LEB128](https://en.wikipedia.org/wiki/LEB128)
  Plain-English explanation of the encoding with worked examples. Use for: building intuition before reading the spec.

- [wabt (WebAssembly Binary Toolkit)](https://github.com/WebAssembly/wabt)
  Source of `wat2wasm`, `wasm-objdump`, `wasm2wat`. Use for: ground-truth diffing — encode the same construct in WAT, run `wat2wasm`, and compare against your own emitter.

- [wazero — embeddable Go WebAssembly runtime](https://github.com/tetratelabs/wazero)
  Pure-Go runtime, no CGo. Use for: instantiating and running the `.wasm` files your backend emits, directly from Go tests.

## Wisdom (Communities)

- [r/ProgrammingLanguages](https://reddit.com/r/ProgrammingLanguages)
  High-signal community for language design and implementation questions.

- [Crafting Interpreters Discord](https://discord.gg/craftinginterpreters)
  Direct community around the book — good for chapter-specific questions.

## Gaps

- No good resource yet for Go-specific AST visitor patterns — may need to synthesise from multiple sources.
