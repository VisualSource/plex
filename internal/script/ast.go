package script

func toAst(tokens []Token) (*Program, error) {
	program := &Program{}
	pos := 0

	for pos < len(tokens) {
		crr := tokens[pos]

		switch crr.IsToken() {
		case TokenType_Keyword:

		case TokenType_Number, TokenType_String, TokenType_Ident:

		default:

		}

	}

	// importStatement
	// ifStatement
	// VariableDeclaration
	// VariableAssignment
	// FunctionDeclaration
	// StructDeclaration
	// whileStatement
	// TernaryStatement
	// Scope
	//
	// Expression
	// Unary ->  - 1, + 1

	return program, nil
}
