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
	Name       string
	Type       AstNode
	Init       AstNode
}

func (v *VariableDeclaration) Range() (Position, Position) {
	return v.Start, v.End
}

type BinaryExpression struct {
	Start, End Position
	Left       AstNode
	Operator   TokenType
	Right      AstNode
}

func (b *BinaryExpression) Range() (Position, Position) {
	return b.Start, b.End
}

type NumberLiteral struct {
	Start, End Position
	Value      string
}

type StringLiteral struct {
	Start, End Position
	Value      string
}

func (s *StringLiteral) Range() (Position, Position) {
	return s.Start, s.End
}

func (m *NumberLiteral) Range() (Position, Position) {
	return m.Start, m.End
}

func NewStringLiteral(value string, start, end Position) *StringLiteral {
	return &StringLiteral{
		Value: value,
		End:   end,
		Start: start,
	}
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

type Type struct {
	Start, End Position
	Name       string
	IsArray    bool
}

func (t *Type) Range() (Position, Position) {
	return t.Start, t.End
}

type Block struct {
	Start, End Position
	Stmts      []AstNode
}

func (b *Block) Range() (Position, Position) {
	return b.Start, b.End
}

type Parameter struct {
	Start, End Position
	Name       string
	Type       AstNode
	Mut        bool
}

func (p *Parameter) Range() (Position, Position) {
	return p.Start, p.End
}

type FunctionDeclaration struct {
	Start, End Position
	Name       string
	Params     []AstNode
	ReturnType AstNode
	Body       AstNode
}

func (f *FunctionDeclaration) Range() (Position, Position) {
	return f.Start, f.End
}

type FunctionCall struct {
	Start, End Position
	Callee     AstNode
	Args       []AstNode
}

func (f *FunctionCall) Range() (Position, Position) {
	return f.Start, f.End
}

type MemberAccess struct {
	Start, End Position
	Object     AstNode
	Field      string
}

func (n *MemberAccess) Range() (Position, Position) {
	return n.Start, n.End
}

type UnaryExpression struct {
	Start, End Position
	Operator   TokenType
	Operand    AstNode
	Postfix    bool
}

type ArrayAccess struct {
	Start, End Position
	Object     AstNode
	Field      AstNode
}

func (a *ArrayAccess) Range() (Position, Position) {
	return a.Start, a.End
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

type IfStatement struct {
	Start, End Position
	Condition  AstNode
	Body       AstNode
	Else       AstNode
}

func (i *IfStatement) Range() (Position, Position) {
	return i.Start, i.End
}

type ImportStatement struct {
	Start, End Position
	Source     string
	Imports    []string
}

func (i *ImportStatement) Range() (Position, Position) {
	return i.Start, i.End
}

type WhileStatement struct {
	Start, End Position
	Condition  AstNode
	Body       AstNode
}

func (w *WhileStatement) Range() (Position, Position) {
	return w.Start, w.End
}

type StructStatement struct {
	Start, End Position
	Name       string
	Fields     []AstNode
}

func (s *StructStatement) Range() (Position, Position) {
	return s.Start, s.End
}

type StructImplStatement struct {
	Start, End Position
	Name       string
	Methods    []AstNode
}

func (s *StructImplStatement) Range() (Position, Position) {
	return s.Start, s.End
}

type ReturnStatement struct {
	Start, End Position
	Value      AstNode
}

func (r *ReturnStatement) Range() (Position, Position) {
	return r.Start, r.End
}
