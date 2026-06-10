# Parser skeleton implemented: peek/advance/expect + basic expression parsing

User implemented the Parser struct with peek, advance, expect and wrote parseExpression/parseTerm correctly — including the grouped expression case `"(" parseExpression() ")"`. They stored Operator as TokenType (not string), and added Range() to BinaryExpression. Test wired up for `1 + 2`.

**Implications:** The grammar→function mechanic is solid. Ready to teach operator precedence — the next natural blocker is that `1 + 2 * 3` will parse wrong with the current flat `parseExpression`.
