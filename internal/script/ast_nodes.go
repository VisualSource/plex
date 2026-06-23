package script

type AstNode interface {
	Range() (Position, Position)
}

type Expression interface {
	AstNode
	GetType() *Type // returns nil if not yet checked
	SetType(*Type)
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
	Type       *TypeExpr
	Init       AstNode
	type_      *Type
}

func (v *VariableDeclaration) GetType() *Type {
	return v.type_
}
func (v *VariableDeclaration) SetType(t *Type) {
	v.type_ = t
}

func (v *VariableDeclaration) Range() (Position, Position) {
	return v.Start, v.End
}

type BinaryExpression struct {
	Start, End Position
	Left       AstNode
	Operator   TokenType
	Right      AstNode
	type_      *Type
}

func (b *BinaryExpression) SetType(t *Type) {
	b.type_ = t
}
func (b *BinaryExpression) GetType() *Type {
	return b.type_
}
func (b *BinaryExpression) Range() (Position, Position) {
	return b.Start, b.End
}

type BooleanLiteral struct {
	Start, End Position
	Value      bool
	type_      *Type
}

func (m *BooleanLiteral) SetType(t *Type) {
	m.type_ = t
}
func (m *BooleanLiteral) GetType() *Type {
	return m.type_
}
func (m *BooleanLiteral) Range() (Position, Position) {
	return m.Start, m.End
}

type NumberLiteral struct {
	Start, End Position
	Value      string
	type_      *Type
}

func (m *NumberLiteral) SetType(t *Type) {
	m.type_ = t
}
func (m *NumberLiteral) GetType() *Type {
	return m.type_
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

type StringLiteral struct {
	Start, End Position
	Value      string
	type_      *Type
}

func (s *StringLiteral) SetType(t *Type) {
	s.type_ = t
}
func (s *StringLiteral) GetType() *Type {
	return s.type_
}
func (s *StringLiteral) Range() (Position, Position) {
	return s.Start, s.End
}
func NewStringLiteral(value string, start, end Position) *StringLiteral {
	return &StringLiteral{
		Value: value,
		End:   end,
		Start: start,
	}
}

type Identifier struct {
	Start, End Position
	Value      string
	type_      *Type
}

func (i *Identifier) SetType(t *Type) {
	i.type_ = t
}
func (i *Identifier) GetType() *Type {
	return i.type_
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

type TypeExpr struct {
	Start, End Position
	Name       string
	IsArray    bool
	ArrayDepth int
	IsNullable bool
	type_      *Type
}

func (t *TypeExpr) GetType() *Type {
	return t.type_
}
func (t *TypeExpr) SetType(v *Type) {
	t.type_ = v
}
func (t *TypeExpr) Range() (Position, Position) {
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
	Type       *TypeExpr
	Mut        bool
	type_      *Type
}

func (p *Parameter) GetType() *Type {
	return p.type_
}
func (p *Parameter) SetType(t *Type) {
	p.type_ = t
}
func (p *Parameter) Range() (Position, Position) {
	return p.Start, p.End
}

type FunctionDeclaration struct {
	Start, End Position
	Name       string
	Params     []*Parameter
	ReturnType *TypeExpr
	Body       AstNode
}

func (f *FunctionDeclaration) Range() (Position, Position) {
	return f.Start, f.End
}

type FunctionCall struct {
	Start, End Position
	Callee     AstNode
	Args       []AstNode
	type_      *Type
}

func (f *FunctionCall) GetType() *Type {
	return f.type_
}
func (f *FunctionCall) SetType(t *Type) {
	f.type_ = t
}
func (f *FunctionCall) Range() (Position, Position) {
	return f.Start, f.End
}

type MemberAccess struct {
	Start, End Position
	Object     AstNode
	Field      string
	type_      *Type
}

func (m *MemberAccess) SetType(t *Type) {
	m.type_ = t
}
func (m *MemberAccess) GetType() *Type {
	return m.type_
}
func (n *MemberAccess) Range() (Position, Position) {
	return n.Start, n.End
}

type UnaryExpression struct {
	Start, End Position
	Operator   TokenType
	Operand    Expression
	Postfix    bool
	type_      *Type
}

func (u *UnaryExpression) GetType() *Type {
	return u.type_
}
func (u *UnaryExpression) SetType(t *Type) {
	u.type_ = t
}
func (u *UnaryExpression) Range() (Position, Position) {
	return u.Start, u.End
}

type ArrayAccess struct {
	Start, End Position

	// what we are indexing with
	Target AstNode

	// what we are indexing into
	Index AstNode
	type_ *Type
}

func (a *ArrayAccess) SetType(t *Type) {
	a.type_ = t
}
func (a *ArrayAccess) GetType() *Type {
	return a.type_
}
func (a *ArrayAccess) Range() (Position, Position) {
	return a.Start, a.End
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
	type_      *Type
}

func (t *TernaryExpression) SetType(n *Type) {
	t.type_ = n
}
func (t *TernaryExpression) GetType() *Type {
	return t.type_
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
	Fields     []*Parameter
}

func (s *StructStatement) Range() (Position, Position) {
	return s.Start, s.End
}

type StructImplStatement struct {
	Start, End Position
	Name       string
	Methods    []*FunctionDeclaration
}

func (s *StructImplStatement) Range() (Position, Position) {
	return s.Start, s.End
}

type ReturnStatement struct {
	Start, End Position
	Value      AstNode
	type_      *Type
}

func (r *ReturnStatement) GetType() *Type {
	return r.type_
}
func (r *ReturnStatement) SetType(t *Type) {
	r.type_ = t
}
func (r *ReturnStatement) Range() (Position, Position) {
	return r.Start, r.End
}

type BreakStatement struct {
	Start, End Position
}

func (r *BreakStatement) Range() (Position, Position) {
	return r.Start, r.End
}

type ContinueStatement struct {
	Start, End Position
}

func (r *ContinueStatement) Range() (Position, Position) {
	return r.Start, r.End
}

type MemberAssignment struct {
	Start, End Position
	Object     AstNode
	Field      string
	Value      AstNode
}

func (m *MemberAssignment) Range() (Position, Position) {
	return m.Start, m.End
}

type ArrayLiteral struct {
	Start, End Position
	Elements   []AstNode
	type_      *Type
}

func (a *ArrayLiteral) GetType() *Type {
	return a.type_
}
func (a *ArrayLiteral) SetType(t *Type) {
	a.type_ = t
}
func (a *ArrayLiteral) Range() (Position, Position) {
	return a.Start, a.End
}

type CastExpression struct {
	Start, End Position
	Expr       AstNode
	TargetType *TypeExpr
	type_      *Type
}

func (c *CastExpression) SetType(t *Type) { c.type_ = t }
func (c *CastExpression) GetType() *Type  { return c.type_ }
func (c *CastExpression) Range() (Position, Position) {
	return c.Start, c.End
}

type ArrayAssignment struct {
	Start, End Position
	Target     *ArrayAccess
	Value      AstNode
}

func (a *ArrayAssignment) Range() (Position, Position) {
	return a.Start, a.End
}
