package typeschecker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/VisualSource/plex/internal/script"
)

type StructInfo struct {
	Fields  map[string]*script.Type
	Methods map[string]*FuncInfo
}

func (s *StructInfo) GetTypeOf(ident string) *script.Type {
	if t, ok := s.Fields[ident]; ok {
		return t
	}

	return nil
}

type FuncArg struct {
	Type *script.Type
	Pos  int
}
type FuncInfo struct {
	Scope      *Scope
	ReturnType *script.Type
	Args       map[string]*FuncArg
}

func newFuncInfo(parent *Scope) *FuncInfo {
	return &FuncInfo{
		Scope: newScope(parent),
		Args:  make(map[string]*FuncArg),
	}
}

type Scope struct {
	parent *Scope
	vars   map[string]*script.Type
}

func newScope(parent *Scope) *Scope {
	return &Scope{
		parent: parent,
		vars:   make(map[string]*script.Type),
	}
}

func (s *Scope) Set(ident string, ty *script.Type) {
	s.vars[ident] = ty
}

func (s *Scope) Get(ident string) *script.Type {
	if t, ok := s.vars[ident]; ok {
		return t
	}

	if s.parent == nil {
		return nil
	}

	return s.parent.Get(ident)
}

type Checker struct {
	structs map[string]*StructInfo
	funcs   map[string]*FuncInfo
	scope   *Scope
	errors  []error

	implBlock      bool
	expectedReturn *script.Type
}

func Check(program *script.Program) error {
	c := &Checker{
		structs: make(map[string]*StructInfo),
		funcs:   make(map[string]*FuncInfo),
		scope:   newScope(nil),
	}

	c.collectSignatures(program)
	return c.checkProgram(program)
}

func (c *Checker) collectSignatures(program script.AstNode) {
	switch n := program.(type) {
	case *script.Program:
		for _, stmt := range n.Stmts {
			c.collectSignatures(stmt)
		}
	case *script.Block:
		for _, s := range n.Stmts {
			c.collectSignatures(s)
		}
	case *script.FunctionDeclaration:
		def := newFuncInfo(c.scope)

		for i, arg := range n.Params {
			def.Args[arg.Name] = &FuncArg{Pos: i}
		}

		c.funcs[n.Name] = def
	case *script.StructStatement:
		if _, ok := c.structs[n.Name]; ok {
			c.error(n, "struct with name %s already exists", n.Name)
			break
		}

		fields := make(map[string]*script.Type)

		c.structs[n.Name] = &StructInfo{
			Fields:  fields,
			Methods: make(map[string]*FuncInfo),
		}

		for _, field := range n.Fields {
			if _, ok := fields[field.Name]; ok {
				c.error(n, "struct %s already has a field named %s", n.Name, field.Name)
				continue
			}

			fields[field.Name] = nil
		}
	case *script.StructImplStatement:
		def, ok := c.structs[n.Name]
		if !ok {
			c.error(n, "struct with name %s already exists", n.Name)
			break
		}

		for _, fn := range n.Methods {
			def.Methods[fn.Name] = newFuncInfo(c.scope)
		}
	}
}
func (c *Checker) checkExpression(expr script.Expression) *script.Type {
	switch n := expr.(type) {
	case *script.Parameter:
		t := c.checkExpression(n.Type)
		n.SetType(t)

		return t
	case *script.TypeExpr:
		var structName string
		var kind script.TypeKind
		switch n.Name {
		case "int":
			kind = script.TypeKind_Int
		case "f64":
			kind = script.TypeKind_F64
		case "i32":
			kind = script.TypeKind_I32
		case "i64":
			kind = script.TypeKind_I64
		case "f32":
			kind = script.TypeKind_F32
		case "float":
			kind = script.TypeKind_Float
		case "string":
			kind = script.TypeKind_String
		case "void":
			kind = script.TypeKind_Void
		default:
			if _, ok := c.structs[n.Name]; ok {
				kind = script.TypeKind_Struct
				structName = n.Name
			} else {

				c.error(n, "unknown type %s", n.Name)
				return nil
			}
		}

		root := &script.Type{
			Kind:   kind,
			Struct: structName,
		}

		if !n.IsArray {
			root.Nullable = n.IsNullable
			n.SetType(root)
			return root
		}

		t := &script.Type{
			Kind:    script.TypeKind_Array,
			Element: root,
		}

		for i := 1; i < n.ArrayDepth; i++ {
			t = &script.Type{
				Kind:    script.TypeKind_Array,
				Element: t,
			}
		}

		t.Nullable = n.IsNullable

		n.SetType(t)
		return t
	case *script.MemberAccess:
		t := c.checkExpression(n.Object.(script.Expression))

		if t == nil {
			return nil
		}

		if t.Kind != script.TypeKind_Struct {
			c.error(n, "cannot access field on non-struct type")
			return nil
		}

		def, ok := c.structs[t.Struct]
		if !ok {
			c.error(n, "no struct with typeof %s", t.Struct)
			return nil
		}

		ft := def.GetTypeOf(n.Field)
		if ft == nil {
			c.error(n, "struct has no field or method with name of %s", n.Field)
			return nil
		}

		n.SetType(ft)

		return ft
	case *script.NumberLiteral:
		var t *script.Type
		if strings.Contains(n.Value, ".") || strings.Contains(n.Value, "e") || strings.Contains(n.Value, "E") {
			t = &script.Type{
				Kind: script.TypeKind_Float,
			}
		} else {
			t = &script.Type{
				Kind: script.TypeKind_Int,
			}
		}

		n.SetType(t)
		return t
	case *script.Identifier:
		ident := c.scope.Get(n.Value)

		if ident == nil {
			c.error(n, "unknown identifier %s", n.Value)
			return nil
		}

		return ident
	case *script.BinaryExpression:
		leftType := c.checkExpression(n.Left.(script.Expression))
		rightType := c.checkExpression(n.Right.(script.Expression))
		if leftType == nil || rightType == nil {
			c.error(n, "failed to resolve left and right types")
			return nil
		}

		if leftType.Kind != rightType.Kind {
			c.error(n, "type for left and right do not match")
			return nil
		}

		n.SetType(leftType)

		return leftType

	default:
		c.error(n, "unhandled node %T", n)
		return nil
	}
}

func isSameType(a, b *script.Type) bool {
	if a == nil || b == nil {
		return false
	}

	return a.Kind == b.Kind && a.Struct == b.Struct
}

func (c *Checker) checkStmt(node script.AstNode) *script.Type {
	switch n := node.(type) {
	case *script.VariableDeclaration:
		t := c.checkStmt(n.Init)

		if n.Type != nil {
			exp := c.checkExpression(n.Type)
			if !isSameType(t, exp) {
				c.error(n, "type annonation does not match init")
				return nil
			}

			c.scope.Set(n.Name, exp)
			n.SetType(exp)
			return exp
		}

		c.scope.Set(n.Name, t)

		n.SetType(t)
		return t
	case *script.Block:
		prev := c.scope
		c.scope = newScope(c.scope)
		for _, stmt := range n.Stmts {
			c.checkStmt(stmt)
		}
		c.scope = prev
	case *script.FunctionDeclaration:

		fn := c.funcs[n.Name]

		fn.ReturnType = c.checkExpression(n.ReturnType)
		if fn.ReturnType == nil {
			fn.ReturnType = &script.Type{
				Kind: script.TypeKind_Void,
			}
		}

		for _, arg := range n.Params {
			t := c.checkExpression(arg)
			fn.Args[arg.Name].Type = t
			fn.Scope.vars[arg.Name] = t
		}

		c.expectedReturn = fn.ReturnType

		prev := c.scope
		c.scope = fn.Scope
		c.checkStmt(n.Body)
		c.scope = prev

		c.expectedReturn = nil
	case *script.IfStatement:
		c.checkStmt(n.Condition)
		c.checkStmt(n.Body)
		if n.Else != nil {
			c.checkStmt(n.Else)
		}
	case *script.WhileStatement:
		c.checkStmt(n.Condition)
		c.checkStmt(n.Body)
	case *script.NumberLiteral:
		return c.checkExpression(n)
	case *script.FunctionCall:

		t := c.checkStmt(n.Callee)

		c.error(n, "unable to")

		return t
	case *script.Identifier:
		return c.checkExpression(n)
	case *script.BinaryExpression:
		return c.checkExpression(n)
	case *script.ReturnStatement:
		rt := c.checkStmt(n.Value)

		if !isSameType(rt, c.expectedReturn) {
			c.error(n, "was expecting return to be %T but was given %T", c.expectedReturn.Kind, rt.Kind)
			return nil
		}

		return rt
	}
	return nil
}

func (c *Checker) checkProgram(program *script.Program) error {
	for _, stmt := range program.Stmts {
		c.checkStmt(stmt)
	}

	if len(c.errors) > 0 {
		return errors.Join(c.errors...)
	}

	return nil
}

func (c *Checker) error(node script.AstNode, format string, args ...any) {
	start, end := node.Range()

	location := fmt.Sprintf("%s\n%s-%s\n", fmt.Sprintf(format, args...), start.String(), end.String())

	c.errors = append(c.errors, fmt.Errorf("%s", location))
}
