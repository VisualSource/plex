# Prior knowledge: tokenizer built, recursive descent understood in theory

User has already implemented a working tokenizer for plex (all token types, whitespace handling, comments, strings, numbers, operators). They understand recursive descent parsing conceptually but have not yet implemented one.

**Implications:** Skip teaching scanning entirely. Start at parsing — specifically at implementing the recursive descent loop and expression parsing. The zone of proximal development is: grammar rules → parse functions → typed AST nodes.
