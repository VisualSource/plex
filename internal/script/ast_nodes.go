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
	Operator    TokenType
}

func (b *BinaryExpression) Range() (Position, Position) {
	return b.Start, b.End
}

type NumberLiteral struct {
	Start, End Position
	Value      string
}

func (m *NumberLiteral) Range() (Position, Position) {
	return m.Start, m.End
}

func NewNumberLiteral(value string, start, end Position) *NumberLiteral {
	return &NumberLiteral{
		Value: value,
		End:   end,
		Start: start,
	}
}

type Identifier struct {
	Start, End Position
	Value      string
}

func (i *Identifier) Range() (Position, Position) {
	return i.Start, i.End
}

func NewIdentifier(value string, start, end Position) *Identifier {
	return &Identifier{
		Value: value,
		Start: start,
		End:   end,
	}
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
	Name       string
	Args       []AstNode
}

func (f *FunctionCall) Range() (Position, Position) {
	return f.Start, f.End
}

type UnaryExpression struct {
	Start, End Position
	Operator   TokenType
	Operand    AstNode
}

func (u *UnaryExpression) Range() (Position, Position) {
	return u.Start, u.End
}

type AssignmentExpression struct {
	Start, End Position
	Name       string
	Value      AstNode
}

func (a *AssignmentExpression) Range() (Position, Position) {
	return a.Start, a.End
}

type TernaryExpression struct {
	Start, End Position
	Condition  AstNode
	TrueBlock  AstNode
	FalseBlock AstNode
}

func (t *TernaryExpression) Range() (Position, Position) {
	return t.Start, t.End
}
