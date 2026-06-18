package typeschecker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/VisualSource/plex/internal/script"
)

type StructInfo struct{}

func (s *StructInfo) GetTypeOf(ident string) *script.Type {
	return nil
}

type FuncInfo struct{}
type Scope struct {
	parent *Scope
	vars   map[string]*script.Type
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
}

func Check(program *script.Program) error {
	c := &Checker{
		structs: make(map[string]*StructInfo),
		funcs:   make(map[string]*FuncInfo),
		scope:   &Scope{vars: make(map[string]*script.Type)},
	}

	c.collectSignatures(program)
	return c.checkProgram(program)
}

func (c *Checker) collectSignatures(program *script.Program) {

}

/*
Typechecker MemberAccess (typecheck.go:59-76)

case *script.MemberAccess:

	t := c.checkExpression(n.Object.(script.Expression))
	def, ok := c.structs[t.Struct]
	...

Blocker — StructInfo is empty (typecheck.go:11-15):

type StructInfo struct{}

	func (s *StructInfo) GetTypeOf(ident string) *script.Type {
	    return nil
	}

This is just a stub. GetTypeOf always returns nil, so every member access will fail with "struct has no field or method with name of X". And collectSignatures is empty too, so c.structs is never populated — even if StructInfo had real data, the map lookup on t.Struct would always miss.

You need:

	type StructInfo struct {
	    Fields  map[string]*script.Type
	    Methods map[string]*FuncInfo  // for method calls later
	}

	func (s *StructInfo) GetTypeOf(ident string) *script.Type {
	    if t, ok := s.Fields[ident]; ok {
	        return t
	    }
	    return nil
	}

And collectSignatures needs to walk the program's StructStatement nodes and populate c.structs[n.Name] with fields. Same shape as prepass in the compiler (compiler.go:736-756).

Bug — nil/kind check missing on t:

t := c.checkExpression(n.Object.(script.Expression))
def, ok := c.structs[t.Struct]   // ← panics if t is nil; silently wrong if t.Kind != Struct
If the recursive check errored, t is nil and dereferencing panics. If the receiver is an i64 literal or similar, t.Struct is "" and you'll get a confusing "no struct with typeof " error. Two short guards:

	if t == nil {
	    return nil  // child already reported
	}

	if t.Kind != script.TypeKind_Struct {
	    c.error(n, "cannot access field on non-struct type")
	    return nil
	}

def, ok := c.structs[t.Struct]
Compiler MemberAccess (compiler.go:467-491)

t := n.Object.(script.Expression).GetType()
def, ok := c.structs[t.Struct]
Good — this is exactly the Task A pattern. Codegen reads the type off the typed AST instead of recomputing. The old typeofStruct call is gone.

Bug — silent fall-through when field not found:

	for _, field := range def.Fields {
	    if field.Name == n.Field {
	        // emit, return nil  ← good
	    }
	}

// ← falls out of switch, returns nil with no error and no WAT emitted
If the field name doesn't match anything, the loop ends and MemberAccess exits cleanly without emitting any WAT. The function returns nil (no error). The caller stitches in nothing, output is broken silently.

In theory the typechecker should have caught this — but the compiler shouldn't trust that. Add a return fmt.Errorf(...) after the loop:

	for _, field := range def.Fields {
	    if field.Name == n.Field {
	        c.emit(fmt.Sprintf("i32.const %d", field.Offset))
	        c.emit("i32.add")
	        c.emit(fmt.Sprintf("%s.load", field.Type))
	        return nil
	    }
	}

return fmt.Errorf("compile: struct %q has no field %q", t.Struct, n.Field)
Same nil-check applies: t := n.Object.(script.Expression).GetType() — if the typechecker didn't fire on this node, t is nil.

Summary
Issue	Severity	Location
StructInfo is empty stub — every field lookup will fail	blocker	typecheck.go:11-15
collectSignatures empty — c.structs never populated	blocker	typecheck.go:53-55
Missing nil/kind guards on t	bug (panics on bad input)	both files
Silent fall-through on field-not-found in compiler	bug (broken output)	compiler.go:484-491
Once StructInfo and collectSignatures are populated, the MemberAccess shape you have will work end-to-end. The compiler side is genuinely correct — it just needs the typechecker upstream to be doing its job.
*/
func (c *Checker) checkExpression(expr script.Expression) *script.Type {
	switch n := expr.(type) {
	case *script.MemberAccess:
		t := c.checkExpression(n.Object.(script.Expression))

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

func (c *Checker) checkStmt(node script.AstNode) *script.Type {
	switch n := node.(type) {
	case *script.Block:
		for _, stmt := range n.Stmts {
			c.checkStmt(stmt)
		}
	case *script.FunctionDeclaration:
		c.checkStmt(n.Body)
	case *script.NumberLiteral:
		return c.checkExpression(n)
	case *script.Identifier:
		return c.checkExpression(n)
	case *script.BinaryExpression:
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
