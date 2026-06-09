package script

type AstNode interface {
	Range() (Position, Position)
}

type Program struct {
	Start, End Position
	Stmts      []AstNode
}

func (p *Program) Range() (Position, Position) {
	return p.Start, p.End
}

type VariableDeclaration struct {
	Start, End Position
	Ident      string
	Init       AstNode
}

func (v *VariableDeclaration) Range() (Position, Position) {
	return v.Start, v.End
}

type BinaryExpression struct {
	Start, End  Position
	Left, Right AstNode
	Operator    string
}

type Scope struct {
	Start, End Position
	Type       rune
	Stmts      []AstNode
}

type Parameter struct {
	Start, End Position
}

type FunctionDeclaration struct {
	Start, End Position
	Params     []Parameter
	Body       Scope
	ReturnType AstNode
}

type FunctionCall struct {
	Start, End Position
	Args       []AstNode
}

type UnaryExpression struct {
	Start, End Position
}
