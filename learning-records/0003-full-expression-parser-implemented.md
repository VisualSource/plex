# Full expression parser implemented with all precedence levels

User implemented the complete expression grammar: primary → postfix → unary → factor → term → comparison → equality → logicAnd → logicOr → ternary → assignment. Also extracted a `binaryExpr` helper that takes the operand-builder and a predicate, eliminating repetition across the binary levels. All postfix cases (call, member access, array index, ++ / --) are handled in a single loop.

**Evidence:** Three passing tests (operator precedence, postfix call, member access). AST nodes for FunctionCall, MemberAccess, ArrayAccess, UnaryExpression, AssignmentExpression, TernaryExpression all defined with Range().

**Implications:** Expression parsing is solid. Ready for statements — the next layer involves dispatching on leading keywords and building the Program node as a list of statements. One subtle bug to surface: parseUnary calls parsePostfix instead of parseUnary recursively, so prefix chains like `--x` won't work.
