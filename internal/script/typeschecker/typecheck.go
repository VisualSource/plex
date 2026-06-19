package typeschecker

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/script"
)

type orderedItem struct {
	Type *script.Type
	Pos  int
}

type StructInfo struct {
	Fields  map[string]*orderedItem
	Methods map[string]*FuncInfo
}

func (s *StructInfo) GetFieldsInOrder() []*orderedItem {
	items := slices.Collect(maps.Values(s.Fields))

	slices.SortFunc(items, func(a, b *orderedItem) int {
		return cmp.Compare(a.Pos, b.Pos)
	})

	return items
}

func (s *StructInfo) GetTypeOf(ident string) *script.Type {
	if t, ok := s.Fields[ident]; ok {
		return t.Type
	}

	return nil
}

type FuncInfo struct {
	Scope      *Scope
	ReturnType *script.Type
	Args       map[string]*orderedItem
}

func (f *FuncInfo) GetArgsInOrder() []*orderedItem {
	items := slices.Collect(maps.Values(f.Args))

	slices.SortFunc(items, func(a, b *orderedItem) int {
		return cmp.Compare(a.Pos, b.Pos)
	})

	return items
}

func newFuncInfo(parent *Scope) *FuncInfo {
	return &FuncInfo{
		Scope: newScope(parent),
		Args:  make(map[string]*orderedItem),
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
	case *script.ImportStatement:
		// import and type check
		switch n.Source {
		case "plex:globals":
			seen := make(map[string]bool)

			for _, imp := range n.Imports {
				switch imp {
				case "heapPtr":
					if _, ok := seen[imp]; ok {
						c.error(n, "already imported %s", imp)
						continue
					}
					seen[imp] = true
					c.scope.Set("heapPtr", &script.Type{
						Kind: script.TypeKind_I32,
					})
				default:
					c.error(n, "unknown import '%s'", imp)

				}
			}
		default:
			c.error(n, "failed to import source file")
		}
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
			def.Args[arg.Name] = &orderedItem{Pos: i}
		}

		c.funcs[n.Name] = def
	case *script.StructStatement:
		if _, ok := c.structs[n.Name]; ok {
			c.error(n, "struct with name %s already exists", n.Name)
			break
		}

		fields := make(map[string]*orderedItem)

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
	type_ := expr.GetType()
	if type_ != nil {
		return type_
	}

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
			c.error(n, "failed to resolve object type for member access")
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
	case *script.StringLiteral:
		t := &script.Type{
			Kind: script.TypeKind_String,
		}

		n.SetType(t)
		return t
	case *script.Identifier:
		ident := c.scope.Get(n.Value)

		if ident == nil {
			c.error(n, "unknown identifier %s", n.Value)
			return nil
		}

		n.SetType(ident)

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
	case *script.ArrayLiteral:
		var expectedType *script.Type

		for _, el := range n.Elements {
			t := c.checkExpression(el.(script.Expression))

			if expectedType == nil {
				expectedType = t
			} else {
				if !isSameType(expectedType, t) {
					c.error(n, "elements do not match in array")
					return nil
				}
			}
		}

		n.SetType(expectedType)

		return expectedType
	case *script.UnaryExpression:
		t := c.checkExpression(n.Operand.(script.Expression))

		n.SetType(t)
		return t
	case *script.ArrayAccess:
		indexType := c.checkExpression(n.Index.(script.Expression))

		if indexType.Kind != script.TypeKind_I64 && indexType.Kind != script.TypeKind_Int {
			c.error(n, "invalid indexing type")
			return nil
		}

		switch target := n.Target.(type) {
		case *script.StringLiteral:
			t := c.checkExpression(target)

			n.SetType(t)
			return t

		case *script.ArrayLiteral:
			t := c.checkExpression(target)

			n.SetType(t.Element)
			return t.Element
		case *script.Identifier:
			i := c.scope.Get(target.Value)

			if i == nil {
				c.error(n, "unknown identifier %s", target.Value)
				return nil
			}

			switch i.Kind {
			case script.TypeKind_Array:
				n.SetType(i.Element)

				return i.Element
			case script.TypeKind_String:
				n.SetType(i)
				return i
			default:
				c.error(n, "unable to index into given target")
				return nil
			}
		default:
			c.error(n, "unable to index into given target")
			return nil
		}
	case *script.FunctionCall:
		switch callee := n.Callee.(type) {
		case *script.MemberAccess:
			revType := c.checkExpression(callee.Object.(script.Expression))
			if revType == nil {
				c.error(n, "failed to resolve type for object")
				return nil
			}
			switch revType.Kind {
			case script.TypeKind_Array:
				switch callee.Field {
				case "append":
					t := &script.Type{
						Kind: script.TypeKind_Void,
					}
					callee.SetType(t)
					n.SetType(t)
					return t
				case "remove":
					c.checkArgs(callee, []*orderedItem{
						{
							Pos: 0,
							Type: &script.Type{
								Kind: script.TypeKind_Int,
							},
						},
					})
					callee.SetType(revType.Element)
					n.SetType(revType.Element)
					return revType.Element
				case "len":
					t := &script.Type{
						Kind: script.TypeKind_Int,
					}
					callee.SetType(t)
					n.SetType(t)
					return t
				default:
					c.error(n, "array has no method %s", callee.Field)
					return nil
				}
			case script.TypeKind_String:
				switch callee.Field {
				case "len":
					t := &script.Type{
						Kind: script.TypeKind_Int,
					}

					callee.SetType(t)
					n.SetType(t)
					return t
				default:
					c.error(n, "array has no method %s", callee.Field)
					return nil
				}
			case script.TypeKind_Struct:
				sdef := c.structs[revType.Struct]
				method, ok := sdef.Methods[callee.Field]
				if !ok {
					c.error(n, "struct %s has no method %s", revType.Struct, callee.Field)
					return nil
				}
				callee.SetType(method.ReturnType)

				c.checkArgs(n, method.GetArgsInOrder())
				n.SetType(method.ReturnType)

				return method.ReturnType
			default:
				c.error(n, "method call on non callable type")
				return nil
			}
		case *script.Identifier:
			if sdef, ok := c.structs[callee.Value]; ok {
				t := &script.Type{Kind: script.TypeKind_Struct, Struct: callee.Value}

				c.checkArgs(n, sdef.GetFieldsInOrder())
				n.SetType(t)
				return t
			}

			if fdef, ok := c.funcs[callee.Value]; ok {
				c.checkArgs(n, fdef.GetArgsInOrder())
				n.SetType(fdef.ReturnType)
				return fdef.ReturnType
			}

			c.error(n, "unknown callable %s", callee.Value)
			return nil
		default:
			c.error(n, "unknown callable %T", callee)
			return nil
		}
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

func (c *Checker) checkArgs(node script.AstNode, args []*orderedItem) {}

func (c Checker) checkFunction(info *FuncInfo, def *script.FunctionDeclaration) {

	if def.ReturnType == nil {
		info.ReturnType = &script.Type{
			Kind: script.TypeKind_Void,
		}
	} else {
		info.ReturnType = c.checkExpression(def.ReturnType)
	}

	for _, arg := range def.Params {
		t := c.checkExpression(arg)
		info.Args[arg.Name].Type = t
		info.Scope.vars[arg.Name] = t
	}

	c.expectedReturn = info.ReturnType

	prev := c.scope
	c.scope = info.Scope
	c.checkStmt(def.Body)
	c.scope = prev

	c.expectedReturn = nil
}

func (c *Checker) checkStmt(node script.AstNode) *script.Type {
	switch n := node.(type) {
	case *script.StructStatement:
		def, ok := c.structs[n.Name]
		if !ok {
			c.error(n, "struct not found")
			return nil
		}

		for i, field := range n.Fields {
			t := c.checkExpression(field)
			def.Fields[field.Name] = &orderedItem{
				Pos:  i,
				Type: t,
			}
		}

		return nil
	case *script.AssignmentExpression:
		v := c.scope.Get(n.Name)

		if v == nil {
			c.error(n, "unknown variable %s", n.Name)
			return nil
		}

		c.checkStmt(n.Value)

	case *script.VariableDeclaration:
		t := c.checkStmt(n.Init)

		if n.Type != nil {
			exp := c.checkExpression(n.Type)
			if !isSameType(t, exp) {
				c.error(n, "type annotation does not match init")
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
		c.checkFunction(fn, n)
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
	case *script.TernaryExpression:
		c.checkStmt(n.Condition)

		a := c.checkStmt(n.FalseBlock)
		b := c.checkStmt(n.TrueBlock)

		if !isSameType(a, b) {
			c.error(n, "left and right no not have matching types")
			return nil
		}

		n.SetType(a)
		return a
	case *script.FunctionCall:
		return c.checkExpression(n)
	case *script.Identifier:
		return c.checkExpression(n)
	case *script.UnaryExpression:
		return c.checkExpression(n)
	case *script.BinaryExpression:
		return c.checkExpression(n)
	case *script.ReturnStatement:
		rt := c.checkStmt(n.Value)

		if (c.expectedReturn == nil || c.expectedReturn.Kind == script.TypeKind_Void) && !(rt == nil || rt.Kind == script.TypeKind_Void) {
			c.error(n, "was expecting return to be void but was given %T", rt.Kind)
			return nil
		} else if !isSameType(rt, c.expectedReturn) {
			if rt != nil {
				c.error(n, "was expecting return to be %T but was given %T", c.expectedReturn.Kind, rt.Kind)
			} else {
				c.error(n, "was expecting return to be %T but not was given", c.expectedReturn.Kind)
			}

			return nil
		}

		n.SetType(rt)

		return rt
	case *script.MemberAssignment:
		t := c.checkExpression(n.Object.(script.Expression))

		def, ok := c.structs[t.Struct]
		if !ok {
			c.error(n, "can not assign to struct %s", t.Struct)
			return nil
		}

		field, ok := def.Fields[n.Field]
		if !ok {
			c.error(n, "unknown field '%s'", n.Field)
			return nil
		}

		vt := c.checkExpression(n.Value.(script.Expression))

		if !isSameType(field.Type, vt) {
			c.error(n, "field type and value do not match")
			return nil
		}

		return vt
	case *script.StructImplStatement:
		def, ok := c.structs[n.Name]
		if !ok {
			c.error(n, "unknown struct")
			return nil
		}

		for _, method := range n.Methods {
			methodDef := def.Methods[method.Name]
			methodDef.Scope.Set("self", &script.Type{
				Kind:   script.TypeKind_Struct,
				Struct: n.Name,
			})

			c.checkFunction(methodDef, method)
		}

	case *script.ArrayAccess:
		return c.checkExpression(n)
	case *script.MemberAccess:
		return c.checkExpression(n)
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
