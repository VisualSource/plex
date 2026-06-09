package script

type AstNode interface{}

func toAst(tokens []Token) ([]AstNode, error) {
	nodes := make([]AstNode, 0)
	pos := 0

	for i := 0; i < len(tokens); i++ {

	}

	// importStatement
	// ifStatement
	// VariableDeclaration
	// VariableAssignment

	// whileStatement
	// Scope
	//
	// Expression
	// Unary ->  - 1, + 1

	return nodes, nil
}
