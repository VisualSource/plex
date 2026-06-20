# Mission: Compiler Construction

## Why
Build the `plex` scripting language end-to-end as a hands-on vehicle for learning how compilers and interpreters actually work — from source text to execution. The language itself is a means to an end: each stage (tokenizer → AST → interpreter/codegen) teaches a foundational compiler concept in a concrete, runnable way.

## Success looks like
- Implement a working recursive-descent parser that produces a typed AST from plex source code
- Walk the AST to evaluate expressions and statements (tree-walking interpreter)
- Understand and explain how each compiler stage maps to theory (grammars, binding, type checking)
- Have a small but complete language you can run real programs in

## Constraints
- Learning in Go (already in use in this repo)
- Building on top of an existing, working tokenizer
- Prefer depth over breadth — understand each stage properly before moving on

## Out of scope
- Code generation / machine code (not yet)
- Optimisation passes
- Tooling (LSP, formatter, etc.)
