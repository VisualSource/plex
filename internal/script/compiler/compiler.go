package compiler

import (
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/script"
	"github.com/VisualSource/plex/internal/script/typeschecker"
)

//go:embed runtime_helpers/*.plex
var globalHelpers embed.FS

type Compiler struct {
	out    strings.Builder
	locals []Local
	depth  int

	structs map[string]*structDef

	globalVars []string

	tempVars int

	implBlock string

	strings   []stringEntry
	dataPtr   int
	needsHeap bool

	inTransaction bool
	transaction   strings.Builder

	labelCount int
	breakStack []string
}

func (c *Compiler) pushBreakLabel() (string, string) {
	bl := fmt.Sprintf("$block%d", c.labelCount)
	ll := fmt.Sprintf("$loop%d", c.labelCount)

	c.labelCount++
	c.breakStack = append(c.breakStack, bl)

	return bl, ll
}
func (c *Compiler) popBreakLabel() {
	c.labelCount--
	c.breakStack = c.breakStack[:len(c.breakStack)-1]
}
func (c *Compiler) currentBreakLabel() string {
	return c.breakStack[len(c.breakStack)-1]
}

func (c *Compiler) startTransaction() {
	c.inTransaction = true
	c.transaction.Reset()

}
func (c *Compiler) endTransaction() {
	c.inTransaction = false
}
func (c *Compiler) writeTransaction() {
	c.out.WriteString(c.transaction.String())
}
func (c *Compiler) emit(parts ...string) {
	var target *strings.Builder
	if c.inTransaction {
		target = &c.transaction
	} else {
		target = &c.out
	}

	target.WriteString(strings.Repeat(" ", c.depth))
	target.WriteString(strings.Join(parts, " "))
	target.WriteRune('\n')
}

func (c *Compiler) Compile(node script.AstNode) error {
	switch n := node.(type) {
	case *script.ImportStatement:
		if c.depth != 1 {
			return fmt.Errorf("can not import in this scope")
		}
	case *script.NumberLiteral:
		t := n.GetType()
		if t == nil {
			return fmt.Errorf("no type set for number")
		}

		prefix := getWasmType(t)

		c.emit(prefix+".const", n.Value)
	case *script.BooleanLiteral:
		switch n.Value {
		case true:
			c.emit("i32.const 1")
			return nil
		case false:
			c.emit("i32.const 0")
			return nil
		}
	case *script.Identifier:

		if !slices.ContainsFunc(c.locals, func(e Local) bool {
			return e.Name == n.Value
		}) && !slices.Contains(c.globalVars, n.Value) {
			return fmt.Errorf("unknown identifier '%s'", n.Value)
		}

		if slices.Contains(c.globalVars, n.Value) {
			c.emit("global.get", "$"+n.Value)
		} else {
			c.emit("local.get", "$"+n.Value)
		}
	case *script.UnaryExpression:
		switch {
		case n.Operator == script.TokenType_Decrement && n.Postfix:
			ident, ok := n.Operand.(*script.Identifier)
			if !ok {
				return errors.ErrUnsupported
			}

			t := ident.GetType()
			wasmType := getWasmType(t)

			c.emit(fmt.Sprintf("local.get $%s", ident.Value))
			c.emit(fmt.Sprintf("local.get $%s", ident.Value))

			c.emit(wasmType + ".const 1")
			c.emit(wasmType + ".sub")

			c.emit(fmt.Sprintf("local.set $%s", ident.Value))

			return nil
		case n.Operator == script.TokenType_Incrment && n.Postfix:
			ident, ok := n.Operand.(*script.Identifier)
			if !ok {
				return errors.ErrUnsupported
			}

			t := ident.GetType()
			wasmType := getWasmType(t)

			c.emit(fmt.Sprintf("local.get $%s", ident.Value))
			c.emit(fmt.Sprintf("local.get $%s", ident.Value))

			c.emit(wasmType + ".const 1")
			c.emit(wasmType + ".add")

			c.emit(fmt.Sprintf("local.set $%s", ident.Value))

		case n.Operator == script.TokenType_Plus && !n.Postfix:
			return c.Compile(n.Operand)
		case n.Operator == script.TokenType_Minus && !n.Postfix:
			t := n.Operand.GetType()
			wasmType := getWasmType(t)

			switch t.Kind {
			case script.TypeKind_F64, script.TypeKind_Float, script.TypeKind_F32:
				if err := c.Compile(n.Operand); err != nil {
					return err
				}
				c.emit(wasmType + ".neg")
			case script.TypeKind_Int, script.TypeKind_I64, script.TypeKind_I32:

				c.emit(wasmType + ".const 0")
				if err := c.Compile(n.Operand); err != nil {
					return err
				}
				c.emit(wasmType + ".sub")
			default:
				return errors.ErrUnsupported
			}

		default:
			return errors.ErrUnsupported
		}
	case *script.BinaryExpression:
		if err := c.Compile(n.Left); err != nil {
			return err
		}
		if err := c.Compile(n.Right); err != nil {
			return err
		}

		t := n.GetType()
		if t == nil {
			return fmt.Errorf("no type set for operation")
		}
		prefix := getWasmType(t)

		switch n.Operator {
		case script.TokenType_Plus:
			c.emit(prefix + ".add")
		case script.TokenType_Minus:
			c.emit(prefix + ".sub")
		case script.TokenType_Star:
			c.emit(prefix + ".mul")
		case script.TokenType_Div:
			divOp := prefix + ".div"
			if prefix == "i32" || prefix == "i64" {
				divOp += "_s"
			}
			c.emit(divOp)
		case script.TokenType_LessThen:
			ltOp := prefix + ".lt"
			if prefix == "i32" || prefix == "i64" {
				ltOp += "_s"
			}
			c.emit(ltOp)
		case script.TokenType_EqualEqual:
			c.emit(prefix + ".eq")
		case script.TokenType_NotEqual:
			c.emit(prefix + ".ne")
		case script.TokenType_GreaterThen:
			ltOp := prefix + ".gt"
			if prefix == "i32" || prefix == "i64" {
				ltOp += "_s"
			}
			c.emit(ltOp)
		case script.TokenType_GreaterThenOrEqaul:
			ltOp := prefix + ".ge"
			if prefix == "i32" || prefix == "i64" {
				ltOp += "_s"
			}
			c.emit(ltOp)
		case script.TokenType_LessThenOrEqual:
			ltOp := prefix + ".le"
			if prefix == "i32" || prefix == "i64" {
				ltOp += "_s"
			}
			c.emit(ltOp)
		case script.TokenType_Mod:
			if prefix == "f32" || prefix == "f64" {
				return fmt.Errorf("unsupported operation on floating point")
			}

			c.emit(prefix + ".rem_s")
		case script.TokenType_AND:
			c.emit("i32.and")
		case script.TokenType_OR:
			c.emit("i32.or")
		default:
			return fmt.Errorf("unsupported operator %v", n.Operator)
		}
	case *script.FunctionDeclaration:
		params := strings.Builder{}
		c.locals = make([]Local, 0)

		skipParam := []string{}

		if c.implBlock != "" {
			skipParam = append(skipParam, "self")
			params.WriteString(" (param $self i32)")
			c.locals = append(c.locals, Local{Name: "self", Type: "i32", Owner: c.implBlock})
		}

		for _, param := range n.Params {
			t := param.GetType()
			if t == nil {
				return fmt.Errorf("no type set on param %s", param.Name)
			}

			wasmType := getWasmType(t)

			params.WriteString(fmt.Sprintf(" (param $%s %s)", param.Name, wasmType))
			c.locals = append(c.locals, Local{Name: param.Name, Type: wasmType, Owner: t.Struct})
			skipParam = append(skipParam, param.Name)
		}

		result := ""
		if n.ReturnType != nil {
			t := n.ReturnType.GetType()
			result = fmt.Sprintf(" (result %s)", getWasmType(t))
		}

		var funcName string
		if c.implBlock != "" {
			funcName = fmt.Sprintf("__%s__%s", c.implBlock, n.Name)
		} else {
			funcName = n.Name
		}

		c.emit(fmt.Sprintf("(func $%s%s%s", funcName, params.String(), result))
		c.depth++

		c.startTransaction()
		//TODO: if theres a return (there should be one if ReturnType is set) validate that the types match ReturnType
		if err := c.Compile(n.Body); err != nil {
			return err
		}
		c.endTransaction()

		for _, local := range c.locals {
			if slices.Contains(skipParam, local.Name) {
				continue
			}
			c.emit(fmt.Sprintf("(local $%s %s)", local.Name, local.Type))
		}

		c.writeTransaction()

		c.depth--
		c.emit(")")

		if c.implBlock == "" {
			// this should be controlled via export keyword not all functions need to be exported
			c.emit(fmt.Sprintf("(export \"%s\" (func $%s))", n.Name, n.Name))
		}

		c.locals = c.locals[:0]

	case *script.ReturnStatement:
		if n.Value != nil {
			if err := c.Compile(n.Value); err != nil {
				return err
			}
		}
		c.emit("return")
	case *script.VariableDeclaration:
		t := n.GetType()

		local := Local{
			Name:  n.Name,
			Type:  getWasmType(t),
			Owner: t.Struct,
		}

		if slices.ContainsFunc(c.locals, func(a Local) bool {
			return a.Name == local.Name
		}) {
			return fmt.Errorf("there is already a variable with name %s", local.Name)
		}

		c.locals = append(c.locals, local)

		if err := c.Compile(n.Init); err != nil {
			return err
		}
		c.emit("local.set $" + n.Name)
	case *script.AssignmentExpression:
		if err := c.Compile(n.Value); err != nil {
			return err
		}

		if slices.Contains(c.globalVars, n.Name) {
			c.emit("global.set $" + n.Name)
		} else {
			c.emit("local.set $" + n.Name)
		}
	case *script.IfStatement:
		if err := c.Compile(n.Condition); err != nil {
			return err
		}
		c.emit("(if")
		c.depth++
		c.emit("(then")
		c.depth++
		if err := c.Compile(n.Body); err != nil {
			return err
		}
		c.depth--
		c.emit(")")

		if n.Else != nil {
			c.emit("(else")
			c.depth++
			if err := c.Compile(n.Else); err != nil {
				return err
			}
			c.depth--
			c.emit(")")
		}

		c.depth--
		c.emit(")")
	case *script.WhileStatement:
		bl, ll := c.pushBreakLabel()
		defer c.popBreakLabel()

		// WASM loops: (block (loop ... br_if 1 ... br 0))
		// br 0 = branch to loop top; br_if 1 = break out of block
		c.emit(fmt.Sprintf("(block %s", bl))
		c.depth++
		c.emit(fmt.Sprintf("(loop %s", ll))
		c.depth++
		if err := c.Compile(n.Condition); err != nil {
			return err
		}

		c.emit("i32.eqz")                   // invert: exit if condition false
		c.emit(fmt.Sprintf("br_if %s", bl)) // jump out of block if condition false

		if err := c.Compile(n.Body); err != nil {
			return err
		}
		c.emit(fmt.Sprintf("br %s", ll)) // jump back to loop top
		c.depth--
		c.emit(")") // end loop
		c.depth--
		c.emit(")") // end block
	case *script.Block:
		for _, stmt := range n.Stmts {
			if err := c.Compile(stmt); err != nil {
				return err
			}

			if expr, ok := stmt.(script.Expression); ok {
				t := expr.GetType()
				if t != nil && t.Kind != script.TypeKind_Void {
					switch stmt.(type) {
					case *script.VariableDeclaration, *script.ReturnStatement:
						// these don't need drop, return is return and variable use local.set
					default:
						c.emit("drop")
					}
				}
			}
		}
	case *script.Program:
		for _, stmt := range n.Stmts {
			if err := c.Compile(stmt); err != nil {
				return err
			}
		}
	case *script.BreakStatement:
		if len(c.breakStack) == 0 {
			return fmt.Errorf("break outside of loop")
		}

		c.emit(fmt.Sprintf("br %s", c.currentBreakLabel()))
	case *script.FunctionCall:
		switch callee := n.Callee.(type) {
		case *script.Identifier:
			if def, ok := c.structs[callee.Value]; ok {
				c.emit(fmt.Sprintf(";; start struct(%s) init", def.Name))

				tmpName := fmt.Sprintf("__struct_%s_tmp%d", def.Name, c.tempVars)
				c.tempVars++

				c.locals = append(c.locals, Local{Name: tmpName, Type: "i32"})

				c.emit(fmt.Sprintf("i32.const %d", def.Size))
				c.emit("call $alloc")
				c.emit(fmt.Sprintf("local.tee $%s", tmpName)) // store addr and leave it on stack

				// init fields

				if len(n.Args) != len(def.Fields) {
					return fmt.Errorf("%d args where passed to ctor when only %d where expected", len(n.Args), len(def.Fields))
				}

				for i, arg := range n.Args {
					field := def.Fields[i]
					c.emit(fmt.Sprintf(";; set field %s", field.Name))
					if i != 0 {
						c.emit(fmt.Sprintf("local.get $%s", tmpName))
					}
					c.emit(fmt.Sprintf("i32.const %d", field.Offset))
					c.emit("i32.add")

					if err := c.Compile(arg); err != nil {
						return err
					}

					c.emit(fmt.Sprintf("%s.store", field.Type))

					c.emit(";; end")
				}
				c.emit(fmt.Sprintf("local.get $%s", tmpName))

				c.emit(fmt.Sprintf(";; end struct(%s) int", def.Name))
			} else {
				for _, arg := range n.Args {
					if err := c.Compile(arg); err != nil {
						return err
					}
				}
				c.emit("call $" + callee.Value)
			}
		case *script.MemberAccess:
			t := callee.Object.(script.Expression).GetType()

			// resolve struct name
			def, ok := c.structs[t.Struct]
			if !ok {
				return fmt.Errorf("compile: unknown struct for member access")
			}

			if _, ok := def.Methods[callee.Field]; !ok {
				return fmt.Errorf("compile: struct %s does not own a method named: %s", t.Struct, callee.Field)
			}

			if err := c.Compile(callee.Object); err != nil { // inject self arg
				return err
			}

			for _, arg := range n.Args {
				if err := c.Compile(arg); err != nil {
					return err
				}
			}

			funcName := fmt.Sprintf("__%s__%s", t.Struct, callee.Field)

			c.emit("call $" + funcName)
		default:
			return fmt.Errorf("compile: unsupported callee %T", n.Callee)
		}
	case *script.StringLiteral:
		for _, entry := range c.strings {
			if entry.value == n.Value {
				c.emit(fmt.Sprintf("i32.const %d", entry.offset))
				return nil
			}
		}
		return fmt.Errorf("compile: string %q not in table", n.Value)

	case *script.ArrayLiteral:
		t := n.GetType()
		if t == nil {
			return fmt.Errorf("array access has no type")
		}
		wasmElem := getWasmType(t.Element)
		elemSize := sizeOf(wasmElem)

		tmpName := fmt.Sprintf("__array_tmp%d", c.tempVars)
		c.tempVars++

		c.locals = append(c.locals, Local{Name: tmpName, Type: "i32"})

		totalSize := 4 + len(n.Elements)*elemSize

		c.emit(";; array literal")

		c.emit(fmt.Sprintf("i32.const %d", totalSize))
		c.emit("call $alloc")
		c.emit(fmt.Sprintf("local.tee $%s", tmpName)) // store addr and leave it on stack

		c.emit(fmt.Sprintf("i32.const %d", len(n.Elements)))
		c.emit("i32.store")

		for i, v := range n.Elements {

			offset := 4 + i*elemSize

			c.emit(fmt.Sprintf(";; insert element %d", i))

			c.emit(fmt.Sprintf("local.get $%s", tmpName))
			c.emit(fmt.Sprintf("i32.const %d", offset))
			c.emit("i32.add")

			if err := c.Compile(v); err != nil {
				return err
			}

			c.emit(wasmElem + ".store")

			c.emit(";; insert end")
		}

		c.emit(fmt.Sprintf("local.get $%s", tmpName)) // put array ptr back on stack

		c.emit(";; end array literal")
	case *script.ArrayAccess:
		t := n.GetType()
		if t == nil {
			return fmt.Errorf("array access has no type")
		}
		wasmElem := getWasmType(t)
		elemSize := sizeOf(wasmElem)

		c.emit(";; array access")

		// get object
		if err := c.Compile(n.Target); err != nil {
			return err
		}
		c.emit("i32.const 4")
		c.emit("i32.add") // skip len header

		if err := c.Compile(n.Index); err != nil {
			return err
		}
		c.emit("i32.wrap_i64") //TODO: need to resolve if we need to do thing

		c.emit(fmt.Sprintf("i32.const %d", elemSize))
		c.emit("i32.mul") // get offset

		c.emit("i32.add")          // base prt + len(4) + offset(i * size)
		c.emit(wasmElem + ".load") // load element

		c.emit(";; end array access")
	case *script.StructStatement:
		// struct should already have been registered
	case *script.MemberAccess:
		if err := c.Compile(n.Object); err != nil {
			return err
		}

		// check what the object is as strings,and arrays have callable methods
		// like len(), append(), remove()

		// TODO: update this so that we can resolve from arrays and the like

		t := n.Object.(script.Expression).GetType()

		// resolve struct name
		def, ok := c.structs[t.Struct]
		if !ok {
			return fmt.Errorf("compile: unknown struct for member access")
		}
		for _, field := range def.Fields {
			if field.Name == n.Field {
				c.emit(fmt.Sprintf("i32.const %d", field.Offset))
				c.emit("i32.add")
				c.emit(fmt.Sprintf("%s.load", field.Type))
				return nil
			}
		}

		return fmt.Errorf("compile: failed to file field %T", node)
	case *script.MemberAssignment:
		if err := c.Compile(n.Object); err != nil {
			return err
		}

		target, ok := n.Object.(script.Expression)
		if !ok {
			return fmt.Errorf("unable to resolve type for member assignment")
		}

		t := target.GetType()
		if t.Kind != script.TypeKind_Struct {
			return fmt.Errorf("assignment on non struct type %s", t)
		}

		// resolve struct name
		def, ok := c.structs[t.Struct]
		if !ok {
			return fmt.Errorf("compile: unknown struct %s for member access", t)
		}
		for _, field := range def.Fields {
			if field.Name == n.Field {
				c.emit(fmt.Sprintf("i32.const %d", field.Offset))
				c.emit("i32.add")

				if err := c.Compile(n.Value); err != nil {
					return err
				}

				c.emit(fmt.Sprintf("%s.store", field.Type))
				return nil
			}
		}
		// load value
	case *script.StructImplStatement:
		def, ok := c.structs[n.Name]
		if !ok {
			return fmt.Errorf("compile: unknown struct")
		}

		c.implBlock = n.Name
		for _, method := range n.Methods {
			methodName := method.Name

			if _, ok := def.Methods[methodName]; ok {
				return fmt.Errorf("struct already has a method named %s", n.Name)
			}
			def.Methods[methodName] = methodName

			if err := c.Compile(method); err != nil {
				return err
			}

		}
		c.implBlock = ""
	case *script.TernaryExpression:
		t := n.GetType()
		if t == nil {
			return fmt.Errorf("ternary: not type set")
		}

		if err := c.Compile(n.Condition); err != nil {
			return err
		}

		wasmType := getWasmType(t)
		c.emit(fmt.Sprintf("(if (result %s)", wasmType))
		c.depth++

		c.emit("(then")
		c.depth++
		if err := c.Compile(n.TrueBlock); err != nil {
			return err
		}
		c.depth--
		c.emit(")")

		c.emit("(else")
		c.depth++
		if err := c.Compile(n.FalseBlock); err != nil {
			return err
		}
		c.depth--
		c.emit(")")

		c.depth--
		c.emit(")")

	default:
		return fmt.Errorf("compile: unhandled %T", node)
	}

	return nil
}

func (c *Compiler) loadHelper(name string) error {
	file, err := globalHelpers.Open("runtime_helpers/" + name)
	if err != nil {
		return err
	}
	defer file.Close()
	helperAst, err := script.Parse(file)
	if err != nil {
		return err
	}

	if err := typeschecker.Check(helperAst); err != nil {
		return err
	}

	if err := c.Compile(helperAst); err != nil {
		return err
	}

	return nil
}

func (c *Compiler) importModules(node *script.Program) error {
	for _, stmt := range node.Stmts {
		switch n := stmt.(type) {
		case *script.ImportStatement:
			if c.depth != 1 {
				return fmt.Errorf("imports must be used at top level")
			}

			switch n.Source {
			case "plex:math":
				for _, imp := range n.Imports {
					switch imp {
					case "sqrt":
						c.emit("(import \"env\" \"sqrt\" (func $sqrt (param if64)))")
					case "pow":
						c.emit("(import \"env\" \"pow\" (func $pow (param f64) (param f64)))")
					default:
						return fmt.Errorf("unknown import: %s", imp)
					}
				}

			case "plex:globals":
				for _, imp := range n.Imports {
					switch imp {
					case "heapPtr":
						c.globalVars = append(c.globalVars, "heapPtr")
						// need to output the code for this at somepoint
					default:
						return fmt.Errorf("unknown import: %s", imp)
					}
				}
			case "plex:console":
				for _, imp := range n.Imports {
					switch imp {
					case "print":
						c.emit("(import \"env\" \"print\" (func $print (param i32)))")
					default:
						return fmt.Errorf("unknown import: %s", imp)
					}
				}
			default:
				return fmt.Errorf("compile: failed to import")
			}
		}
	}

	return nil
}

// collect static strings
// check for heap objects (structs,arrays, maybe strings?)
// register structs
func (c *Compiler) prepass(node script.AstNode) {
	switch n := node.(type) {
	case *script.Program:
		for _, child := range n.Stmts {
			c.prepass(child)
		}
	case *script.Block:
		for _, child := range n.Stmts {
			c.prepass(child)
		}
	case *script.FunctionDeclaration:
		c.prepass(n.Body)
	case *script.IfStatement:
		c.prepass(n.Condition)
		c.prepass(n.Body)
		if n.Else != nil {
			c.prepass(n.Else)
		}
	case *script.WhileStatement:
		c.prepass(n.Condition)
		c.prepass(n.Body)
	case *script.ReturnStatement:
		c.prepass(n.Value)
	case *script.VariableDeclaration:
		c.prepass(n.Init)
	case *script.FunctionCall:
		for _, arg := range n.Args {
			c.prepass(arg)
		}
	case *script.StringLiteral:
		c.strings = append(c.strings, stringEntry{
			value:  n.Value,
			offset: c.dataPtr,
		})
		c.dataPtr += len(n.Value) + 1
	case *script.ArrayLiteral:
		c.needsHeap = true
		for _, arg := range n.Elements {
			c.prepass(arg)
		}
	case *script.StructStatement:
		c.needsHeap = true

		def := &structDef{Name: n.Name, Methods: make(map[string]string), Fields: make([]structField, 0), Size: 0}

		for _, param := range n.Fields {
			t := param.GetType()

			wasmType := getWasmType(t)
			size := sizeOf(wasmType)

			def.Fields = append(def.Fields, structField{
				Name:   param.Name,
				Type:   wasmType,
				Offset: def.Size,
			})

			def.Size += size

		}

		c.structs[n.Name] = def

	}
}

func CompileProgram(program *script.Program) (string, error) {
	if err := typeschecker.Check(program); err != nil {
		return "", err
	}

	c := &Compiler{
		structs: make(map[string]*structDef),
	}
	c.prepass(program)

	c.emit("(module")
	c.depth++

	c.importModules(program)
	// collect struct/arrays/ dyn strings

	if c.needsHeap || len(c.strings) > 0 {
		c.emit("(memory 1)")

		for _, entry := range c.strings {
			escaped := strings.ReplaceAll(entry.value, "\\", "\\\\")
			escaped = strings.ReplaceAll(escaped, "\"", "\\\"")

			c.emit(fmt.Sprintf("(data (i32.const %d) \"%s\\00\")", entry.offset, escaped))
		}

		c.emit("(export \"memory\" (memory 0))")

		if c.needsHeap {

			// round this, if 1000 -> 1024
			heapStart := int(math.Ceil(float64(c.dataPtr)/8) * 8)
			c.emit(fmt.Sprintf("(global $heapPtr (mut i32) (i32.const %d))", heapStart))
			c.globalVars = append(c.globalVars, "heapPtr")
			//inject helper
			if err := c.loadHelper("global.plex"); err != nil {
				return "", err
			}
		}

	}

	for _, stmt := range program.Stmts {
		if err := c.Compile(stmt); err != nil {
			return "", err
		}
	}
	c.depth--
	c.emit(")")
	return c.out.String(), nil
}
