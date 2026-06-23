package typeschecker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/VisualSource/plex/internal/script"
)

type Checker struct {
	structs map[string]*StructInfo
	funcs   map[string]*FuncInfo
	scope   *Scope
	errors  []error

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
		importModule(c, n)
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
			c.error(n, fmt.Errorf("struct with name %s already exists", n.Name))
			break
		}

		fields := make(map[string]*orderedItem)

		c.structs[n.Name] = &StructInfo{
			Fields:  fields,
			Methods: make(map[string]*FuncInfo),
		}

		for _, field := range n.Fields {
			if _, ok := fields[field.Name]; ok {
				c.error(n, fmt.Errorf("struct %s already has a field named %s", n.Name, field.Name))
				continue
			}

			fields[field.Name] = nil
		}
	case *script.StructImplStatement:
		def, ok := c.structs[n.Name]
		if !ok {
			c.error(n, fmt.Errorf("struct with name %s already exists", n.Name))
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
		case "bool":
			kind = script.TypeKind_Bool
		default:
			if _, ok := c.structs[n.Name]; ok {
				kind = script.TypeKind_Struct
				structName = n.Name
			} else {

				c.error(n, fmt.Errorf("unknown type %s", n.Name))
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
			c.error(n, errors.New("failed to resolve object type for member access"))
			return nil
		}

		if t.Kind != script.TypeKind_Struct {
			c.error(n, errors.New("cannot access field on non-struct type"))
			return nil
		}

		def, ok := c.structs[t.Struct]
		if !ok {
			c.error(n, fmt.Errorf("no struct with typeof %s", t.Struct))
			return nil
		}

		ft := def.GetTypeOf(n.Field)
		if ft == nil {
			c.error(n, fmt.Errorf("struct has no field or method with name of %s", n.Field))
			return nil
		}

		n.SetType(ft)

		return ft
	case *script.BooleanLiteral:
		t := &script.Type{
			Kind: script.TypeKind_Bool,
		}

		n.SetType(t)

		return t
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
			c.error(n, fmt.Errorf("unknown identifier %s", n.Value))
			return nil
		}

		n.SetType(ident)

		return ident
	case *script.BinaryExpression:
		leftType := c.checkExpression(n.Left.(script.Expression))
		rightType := c.checkExpression(n.Right.(script.Expression))
		if !isSameType(leftType, rightType) {
			c.error(n, errors.New("type for left and right do not match"))
			return nil
		}

		switch n.Operator {
		case script.TokenType_AND, script.TokenType_OR,
			script.TokenType_EqualEqual, script.TokenType_NotEqual,
			script.TokenType_LessThen, script.TokenType_GreaterThen,
			script.TokenType_LessThenOrEqual, script.TokenType_GreaterThenOrEqaul:
			t := &script.Type{
				Kind: script.TypeKind_Bool,
			}

			n.SetType(t)
			return t
		default:
			n.SetType(leftType)
			return leftType
		}
	case *script.ArrayLiteral:
		var innerType *script.Type

		for _, el := range n.Elements {
			t := c.checkExpression(el.(script.Expression))

			if innerType == nil {
				innerType = t
			} else {
				if !isSameType(innerType, t) {
					c.error(n, errors.New("elements do not match in array"))
					return nil
				}
			}
		}

		t := &script.Type{
			Kind:    script.TypeKind_Array,
			Element: innerType,
		}

		n.SetType(t)
		return t
	case *script.UnaryExpression:
		t := c.checkExpression(n.Operand)

		n.SetType(t)
		return t
	case *script.ArrayAccess:
		indexType := c.checkExpression(n.Index.(script.Expression))

		if indexType.Kind != script.TypeKind_I64 && indexType.Kind != script.TypeKind_Int {
			c.error(n, fmt.Errorf("can not index using type %d", indexType.Kind))
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
				c.error(n, fmt.Errorf("unknown identifier %s", target.Value))
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
				c.error(n, errors.New("unable to index into given target"))
				return nil
			}
		default:
			c.error(n, errors.New("unable to index into given target"))
			return nil
		}
	case *script.FunctionCall:
		switch callee := n.Callee.(type) {
		case *script.MemberAccess:
			revType := c.checkExpression(callee.Object.(script.Expression))
			if revType == nil {
				c.error(n, errors.New("failed to resolve type for object"))
				return nil
			}
			switch revType.Kind {
			case script.TypeKind_Array:
				switch callee.Field {
				case "append":
					if !c.checkArgs(n, []*orderedItem{
						{
							Pos:  0,
							Type: revType.Element,
						},
					}) {
						return nil
					}

					t := &script.Type{
						Kind: script.TypeKind_Void,
					}
					callee.SetType(t)
					n.SetType(t)
					return t
				case "remove":
					if !c.checkArgs(n, []*orderedItem{
						{
							Pos: 0,
							Type: &script.Type{
								Kind: script.TypeKind_Int,
							},
						},
					}) {
						return nil
					}

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
					c.error(n, fmt.Errorf("array has no method %s", callee.Field))
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
					c.error(n, fmt.Errorf("array has no method %s", callee.Field))
					return nil
				}
			case script.TypeKind_Struct:
				sdef := c.structs[revType.Struct]
				method, ok := sdef.Methods[callee.Field]
				if !ok {
					c.error(n, fmt.Errorf("struct %s has no method %s", revType.Struct, callee.Field))
					return nil
				}
				callee.SetType(method.ReturnType)

				if !c.checkArgs(n, method.GetArgsInOrder()) {
					return nil
				}
				n.SetType(method.ReturnType)

				return method.ReturnType
			default:
				c.error(n, errors.New("method call on non callable type"))
				return nil
			}
		case *script.Identifier:
			if sdef, ok := c.structs[callee.Value]; ok {
				t := &script.Type{Kind: script.TypeKind_Struct, Struct: callee.Value}

				if !c.checkArgs(n, sdef.GetFieldsInOrder()) {
					return nil
				}
				n.SetType(t)
				return t
			}

			if fdef, ok := c.funcs[callee.Value]; ok {
				if !c.checkArgs(n, fdef.GetArgsInOrder()) {
					return nil
				}
				n.SetType(fdef.ReturnType)
				return fdef.ReturnType
			}

			c.error(n, fmt.Errorf("unknown callable %s", callee.GetType()))
			return nil
		default:
			c.error(n, fmt.Errorf("unknown callable"))
			return nil
		}
	case *script.CastExpression:
		targetType := c.checkExpression(n.TargetType)
		if targetType == nil {
			c.error(n, errors.New("failed to resolve type"))
			return nil
		}

		srcType := c.checkStmt(n.Expr)
		if srcType == nil {
			c.error(n, errors.New("failed to resolve type"))
			return nil
		}

		if !isNumeric(srcType) || !isNumeric(targetType) {
			c.error(n, fmt.Errorf(
				"cannot cast %s to %s: only numeric types are castable",
				srcType,
				targetType,
			))
			return nil
		}

		n.SetType(targetType)
		return targetType
	default:
		c.error(n, fmt.Errorf("unhandled node %T", n))
		return nil
	}
}

func (c *Checker) checkArgs(node *script.FunctionCall, args []*orderedItem) bool {
	if len(node.Args) != len(args) {
		c.error(node, fmt.Errorf("was expecting %d args but was given %d", len(args), len(node.Args)))

		return false
	}

	errors := true
	for _, expected := range args {
		want := node.Args[expected.Pos]
		t := c.checkExpression(want.(script.Expression))

		if !isSameType(expected.Type, t) {
			errors = false

			c.error(want, fmt.Errorf("was expecting %s but was given %s", expected.Type.Kind, t.Kind))
		}
	}

	return errors
}

func (c *Checker) checkFunction(info *FuncInfo, def *script.FunctionDeclaration) {

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
	case *script.ImportStatement:
		return nil
	case *script.StructStatement:
		def, ok := c.structs[n.Name]
		if !ok {
			c.error(n, ErrUnknownTypename)
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
			c.error(n, fmt.Errorf("unknown variable %s", n.Name))
			return nil
		}

		c.checkStmt(n.Value)
		return nil
	case *script.VariableDeclaration:
		t := c.checkStmt(n.Init)

		if n.Type != nil {
			exp := c.checkExpression(n.Type)
			if !isSameType(t, exp) {
				c.error(n, errors.New("type annotation does not match init"))
				return nil
			}

			c.scope.Set(n.Name, exp)
			n.SetType(exp)
			return exp
		}

		if t == nil {
			t = &script.Type{
				Kind: script.TypeKind_Unknown,
			}
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
		return nil
	case *script.FunctionDeclaration:
		fn := c.funcs[n.Name]
		c.checkFunction(fn, n)
		return nil
	case *script.IfStatement:
		c.checkStmt(n.Condition)
		c.checkStmt(n.Body)
		if n.Else != nil {
			c.checkStmt(n.Else)
		}
		return nil
	case *script.WhileStatement:
		c.checkStmt(n.Condition)
		c.checkStmt(n.Body)
		return nil
	case *script.NumberLiteral:
		return c.checkExpression(n)
	case *script.TernaryExpression:
		c.checkStmt(n.Condition)

		a := c.checkStmt(n.FalseBlock)
		b := c.checkStmt(n.TrueBlock)

		if !isSameType(a, b) {
			c.error(n, errors.New("left and right no not have matching types"))
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
			c.error(n, fmt.Errorf("was expecting return to be void but was given %s", rt.Kind))
			return nil
		} else if !isSameType(rt, c.expectedReturn) {
			if rt != nil {
				c.error(n, fmt.Errorf("was expecting return to be %s but was given %s", c.expectedReturn, rt))
			} else {
				c.error(n, fmt.Errorf("was expecting return to be %s but not was given", c.expectedReturn))
			}

			return nil
		}

		n.SetType(rt)

		return rt
	case *script.MemberAssignment:
		t := c.checkExpression(n.Object.(script.Expression))

		def, ok := c.structs[t.Struct]
		if !ok {
			c.error(n, fmt.Errorf("can not assign to struct %s", t.Struct))
			return nil
		}

		field, ok := def.Fields[n.Field]
		if !ok {
			c.error(n, fmt.Errorf("unknown field '%s'", n.Field))
			return nil
		}

		vt := c.checkExpression(n.Value.(script.Expression))

		if !isSameType(field.Type, vt) {
			c.error(n, errors.New("field type and value do not match"))
			return nil
		}

		return vt
	case *script.StructImplStatement:
		def, ok := c.structs[n.Name]
		if !ok {
			c.error(n, ErrUnknownTypename)
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
		return nil
	case *script.BooleanLiteral:
		return c.checkExpression(n)
	case *script.ArrayLiteral:
		return c.checkExpression(n)
	case *script.ArrayAccess:
		return c.checkExpression(n)
	case *script.MemberAccess:
		return c.checkExpression(n)
	case *script.StringLiteral:
		return c.checkExpression(n)
	case *script.BreakStatement:
		return nil
	case *script.ContinueStatement:
		return nil
	case *script.CastExpression:
		return c.checkExpression(n)
	default:
		c.error(n, fmt.Errorf("unhandled node %T", n))
		return nil
	}
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

func (c *Checker) error(node script.AstNode, reason error) {
	te := NewTypeError(node, reason)

	c.errors = append(c.errors, te)
}
