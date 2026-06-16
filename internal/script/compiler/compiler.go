package compiler

import (
	"embed"
	_ "embed"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/script"
)

//go:embed runtime_helpers/*.plex
var globalHelpers embed.FS

type Compiler struct {
	out    strings.Builder
	locals []Local
	depth  int

	structs map[string]structDef

	globalVars []string

	tempVars int

	strings   []stringEntry
	dataPtr   int
	needsHeap bool

	inTransaction bool
	transaction   strings.Builder
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
		c.emit("f64.const", n.Value)
	case *script.Identifier:
		if slices.Contains(c.globalVars, n.Value) {
			c.emit("global.get", "$"+n.Value)
		} else {
			c.emit("local.get", "$"+n.Value)
		}
	case *script.BinaryExpression:
		if err := c.Compile(n.Left); err != nil {
			return err
		}
		if err := c.Compile(n.Right); err != nil {
			return err
		}

		tL, err := c.typeOf(n.Left)
		if err != nil {
			return err
		}
		tR, err := c.typeOf(n.Right)
		if err != nil {
			return err
		}

		prefix := "unknown"
		if tL == "unknown" && tR != "unknown" {
			prefix = tR
		} else if tL != "unknown" && tR == "unknown" {
			prefix = tL
		} else if tL == tR && tL != "unknown" {
			prefix = tL
		} else {
			return fmt.Errorf("type miss match %s %s", tL, tR)
		}

		//TODO some type inpection so we can find out what we need to use here
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
		default:
			return fmt.Errorf("unsupported operator %v", n.Operator)
		}
	case *script.FunctionDeclaration:
		params := strings.Builder{}
		c.locals = make([]Local, 0)

		for _, p := range n.Params {
			param := p.(*script.Parameter)

			t := resolveType(param.Type.(*script.Type))

			params.WriteString(fmt.Sprintf(" (param $%s %s)", param.Name, t))
			c.locals = append(c.locals, Local{Name: param.Name, Type: t})
		}

		result := ""
		if n.ReturnType != nil {
			t := resolveType(n.ReturnType)
			result = fmt.Sprintf(" (result %s)", t)
		}

		c.emit(fmt.Sprintf("(func $%s%s%s", n.Name, params.String(), result))
		c.depth++

		c.startTransaction()
		//TODO: if theres a return (there should be one if ReturnType is set) validate that the types match ReturnType
		if err := c.Compile(n.Body); err != nil {
			return err
		}
		c.endTransaction()

		for _, local := range c.locals {
			c.emit(fmt.Sprintf("(local $%s %s)", local.Name, local.Type))
		}

		c.writeTransaction()

		c.depth--
		c.emit(")")

		// this should be controlled via export keyword not all functions need to be exported
		c.emit(fmt.Sprintf("(export \"%s\" (func $%s))", n.Name, n.Name))

		c.locals = c.locals[:0]

	case *script.ReturnStatement:
		if n.Value != nil {
			if err := c.Compile(n.Value); err != nil {
				return err
			}
		}
		c.emit("return")
	case *script.VariableDeclaration:
		local := Local{
			Name: n.Name,
			Type: "f64", // f64,i64,f32,i32
		}

		if n.Type != nil { // use explict type
			local.Type = resolveType(n.Type)
			//TODO: should valiate the declaration expresion matchs explict type
		} else {
			// implicit type
			switch i := n.Init.(type) {
			case *script.NumberLiteral:
				// TODO: should reslove to i64 by default or f64 when thers a . in the number
			case *script.FunctionCall:
				local.Type = "i32"

				switch callee := i.Callee.(type) {
				case *script.Identifier:
					if def, ok := c.structs[callee.Value]; ok {
						local.Owner = def.Name
					}
				default:
					// todo reslove type from function, throw error if number returns void
					return fmt.Errorf("unable to reslove type")
				}

			default: // arrays, structs, strings will be i32 for points
				local.Type = "i32"
			}
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
		c.emit("if")
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
		// WASM loops: (block (loop ... br_if 1 ... br 0))
		// br 0 = branch to loop top; br_if 1 = break out of block
		c.emit("(block")
		c.depth++
		c.emit("(loop")
		c.depth++
		if err := c.Compile(n.Condition); err != nil {
			return err
		}
		c.emit("i32.eqz") // invert: exit if condition false
		c.emit("br_if 1") // jump out of block if condition false

		if err := c.Compile(n.Body); err != nil {
			return err
		}
		c.emit("br 0") // jump back to loop top
		c.depth--
		c.emit(")") // end loop
		c.depth--
		c.emit(")") // end block
	case *script.Block:
		for _, stmt := range n.Stmts {
			if err := c.Compile(stmt); err != nil {
				return err
			}
		}
	case *script.Program:
		for _, stmt := range n.Stmts {
			if err := c.Compile(stmt); err != nil {
				return err
			}
		}
	case *script.FunctionCall:
		for _, arg := range n.Args {
			if err := c.Compile(arg); err != nil {
				return err
			}
		}

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

					c.emit(fmt.Sprintf("local.get $%s", tmpName))
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
				c.emit("call $" + callee.Value)
			}
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

		tmpName := fmt.Sprintf("__array_tmp%d", c.tempVars)
		c.tempVars++

		c.locals = append(c.locals, Local{Name: tmpName, Type: "i32"})

		elemSize := 8 // f64 element size in bytes, would need to reslove size for structs
		totalSize := 4 + len(n.Elements)*elemSize

		// allocate memory, capture base in a temp loca

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

			c.emit("f64.store")

			c.emit(";; insert end")
		}

		c.emit(fmt.Sprintf("local.get $%s", tmpName)) // put array ptr back on stack

		c.emit(";; end array literal")
	case *script.ArrayAccess:

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

		c.emit("i32.const 8") // 8 = f64, should be size of object ex. sizeof(struct) || sizeof(string)
		c.emit("i32.mul")     // get offset

		c.emit("i32.add")  // base prt + len(4) + offset(i * size)
		c.emit("f64.load") // load element, TODO: resolve TYPE HERE

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
		id, ok := n.Object.(*script.Identifier)
		if !ok {
			return fmt.Errorf("unable to resolve struct")
		}

		// resolve struct name
		def, ok := c.structs[id.Value]
		if !ok {
			return fmt.Errorf("compile: unknown struct %q for member access", id.Value)
		}
		for _, field := range def.Fields {
			if field.Name == n.Field {
				c.emit(fmt.Sprintf("i32.const %d", field.Offset))
				c.emit("i32.add")
				c.emit(fmt.Sprintf("%s.load", field.Type))
				return nil
			}
		}
	case *script.MemberAssignment:
		if err := c.Compile(n.Object); err != nil {
			return err
		}

		// TODO: update this so that we can resolve from arrays and the like
		id, ok := n.Object.(*script.Identifier)
		if !ok {
			return fmt.Errorf("unable to resolve struct")
		}

		// resolve struct name
		def, ok := c.structs[id.Value]
		if !ok {
			return fmt.Errorf("compile: unknown struct %q for member access", id.Value)
		}
		for _, field := range def.Fields {
			if field.Name == n.Field {
				c.emit(fmt.Sprintf("i32.const %d", field.Offset))
				c.emit("i32.add")
				c.emit(fmt.Sprintf("%s.store", field.Type))
				return nil
			}
		}
		// load value
	default:
		return fmt.Errorf("compile: unhandled %T", node)
	}

	return nil
}

func (c *Compiler) typeOf(node script.AstNode) (string, error) {
	switch n := node.(type) {
	case *script.NumberLiteral:
		if strings.Contains(n.Value, ".") {
			return "f64", nil
		}
		return "i64", nil
	case *script.Identifier:
		for _, l := range c.locals {
			if l.Name == n.Value {
				return l.Type, nil
			}
		}
		return "unknown", nil
	case *script.BinaryExpression:
		tl, err := c.typeOf(n.Left)
		if err != nil {
			return "", err
		}
		tr, err := c.typeOf(n.Right)
		if err != nil {
			return "", err
		}

		if tl == "unknown" && tr == "unknown" {
			return "unknown", nil
		}

		if tl == "unknown" && tr != "unknown" {
			return tr, nil
		}

		if tl != "unknown" && tr == "unknown" {
			return tl, nil
		}

		if tl != tr {
			return "", fmt.Errorf("type %s does not match %s", tl, tr)
		}

		return tl, nil
	case *script.FunctionCall:
		return "f64", nil //TODO: look up return from function type
	default:
		return "unknown", nil
	}
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
			case "plex:globals":
				for _, imp := range n.Imports {
					switch imp {
					case "heapPtr":
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

		def := structDef{Name: n.Name, Fields: make([]structField, 0), Size: 0}

		offset := 0
		for _, f := range n.Fields {
			param := f.(*script.Parameter)
			wasmType := resolveType(param.Type)
			size := sizeOf(wasmType)

			def.Fields = append(def.Fields, structField{
				Name:   param.Name,
				Type:   wasmType,
				Offset: offset,
			})

			offset += size
		}

		c.structs[n.Name] = def

	}
}

func CompileProgram(program *script.Program) (string, error) {
	c := &Compiler{
		structs: make(map[string]structDef),
	}
	c.prepass(program)

	c.emit("(module")
	c.depth++

	c.importModules(program)

	if len(c.strings) > 0 {
		c.emit("(memory 1)")

		for _, entry := range c.strings {
			escaped := strings.ReplaceAll(entry.value, "\\", "\\\\")
			escaped = strings.ReplaceAll(escaped, "\"", "\\\"")

			c.emit(fmt.Sprintf("(data (i32.const %d) \"%s\\00\")", entry.offset, escaped))
		}

		c.emit("(export \"memory\" (memory 0))")
	}

	// collect struct/arrays/ dyn strings

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

	for _, stmt := range program.Stmts {
		if err := c.Compile(stmt); err != nil {
			return "", err
		}
	}
	c.depth--
	c.emit(")")
	return c.out.String(), nil
}
