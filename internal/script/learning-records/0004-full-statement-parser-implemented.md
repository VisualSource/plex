# Full statement parser implemented, including impl blocks and return

User implemented all statement forms: import (multi-name), let/var decl, if/else, while, fn, struct, impl (struct method blocks), return. Added Block, Type, Parameter, all statement AST nodes with Range(). Also introduced parseStructImpl unprompted — went beyond the lesson scope.

**Bug present:** isKeyword checks TokenType_Keyword but tokenizer emits all identifiers including keywords as TokenType_Ident. Keyword dispatch silently falls through to parseExprStmt. Tests don't catch it because no existing test exercises a keyword-led statement.

**Implications:** Ready for the interpreter. Fix keyword bug first (tokenizer consumeIdent should emit keywords as TokenType_Keyword). Then Lesson 4: tree-walking evaluation — Value types, Environment scope chain, Eval dispatcher, return propagation as a sentinel error.
