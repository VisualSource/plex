package script

type AstNode interface{}

func toAst(tokens []Token) ([]AstNode, error) {
	nodes := make([]AstNode, 0)
	pos := 0

	for pos < len(tokens) {

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

	return nodes, nil
}
