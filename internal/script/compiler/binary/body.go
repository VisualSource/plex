package binary_wasm

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"

	"github.com/VisualSource/plex/internal/script"
)

type bodyEncoder struct {
	buf        []byte
	locals     map[string]uint32
	localTypes []byte

	blockDepth int
	loopStack  []loopEntry

	funcIndices map[string]uint32
}

func newBodyEncoder(f *script.FunctionDeclaration, sig funcSig, funcIndices map[string]uint32) *bodyEncoder {
	b := &bodyEncoder{locals: make(map[string]uint32), funcIndices: funcIndices}
	var idx uint32
	if sig.isMethod {
		b.locals["self"] = idx
		idx++
	}
	for _, p := range f.Params {
		b.locals[p.Name] = idx
		idx++
	}
	b.collectLocals(f.Body)

	return b
}

func (b *bodyEncoder) walk(node script.AstNode) error {
	switch n := node.(type) {
	case *script.Block:
		for _, s := range n.Stmts {
			if err := b.walk(s); err != nil {
				return err
			}
			if expr, ok := s.(script.Expression); ok {
				t := expr.GetType()
				if t != nil && t.Kind != script.TypeKind_Void {
					switch s.(type) {
					case *script.VariableDeclaration, *script.AssignmentExpression, *script.ReturnStatement:
					default:
						b.buf = append(b.buf, OpDrop)
					}
				}
			}
		}
		return nil
	case *script.BooleanLiteral:
		b.buf = append(b.buf, OpI32Const)
		if n.Value {
			b.buf = append(b.buf, 1)
		} else {
			b.buf = append(b.buf, 0)
		}
		return nil
	case *script.NumberLiteral:
		t := n.GetType()
		if t == nil {
			return fmt.Errorf("no type on number %s", n.Value)
		}
		switch t.Kind {
		case script.TypeKind_I32:
			v, _ := strconv.ParseInt(n.Value, 10, 32)
			b.buf = append(b.buf, OpI32Const)
			b.buf = AppendSLEB128(b.buf, v)
		case script.TypeKind_I64, script.TypeKind_Int:
			v, _ := strconv.ParseInt(n.Value, 10, 64)
			b.buf = append(b.buf, OpI64Const)
			b.buf = AppendSLEB128(b.buf, v)
		case script.TypeKind_F64, script.TypeKind_Float:
			v, _ := strconv.ParseFloat(n.Value, 64)
			b.buf = append(b.buf, OpF64Const)
			var raw [8]byte
			binary.LittleEndian.PutUint64(raw[:], math.Float64bits(v))
			b.buf = append(b.buf, raw[:]...)
		default:
			return fmt.Errorf("unsupported number type %s", t)
		}
		return nil
	case *script.ReturnStatement:
		if n.Value != nil {
			if err := b.walk(n.Value); err != nil {
				return err
			}
		}
		b.buf = append(b.buf, OpReturn)
		return nil
	case *script.Identifier:
		idx, ok := b.locals[n.Value]
		if !ok {
			return fmt.Errorf("unknown identifier %s", n.Value)
		}
		b.buf = append(b.buf, OpLocalGet)
		b.buf = AppendULEB128(b.buf, idx)
		return nil
	case *script.BinaryExpression:
		if n.Operator == script.TokenType_AND || n.Operator == script.TokenType_OR {
			return b.walkShortCircuit(n)
		}

		if err := b.walk(n.Left); err != nil {
			return err
		}
		if err := b.walk(n.Right); err != nil {
			return err
		}
		leftExpr, ok := n.Left.(script.Expression)
		if !ok {
			return fmt.Errorf("binary expression left has no type info")
		}
		opType := leftExpr.GetType()
		if opType == nil {
			return fmt.Errorf("no type on binary expression operand")
		}

		if isCmpOp(n.Operator) {
			op, err := cmpOpcode(opType.Kind, n.Operator)
			if err != nil {
				return err
			}
			b.buf = append(b.buf, op)
			return nil
		}

		op, err := arithOpcode(opType.Kind, n.Operator)
		if err != nil {
			return err
		}
		b.buf = append(b.buf, op)
		return nil
	case *script.VariableDeclaration:
		if n.Init != nil {
			if err := b.walk(n.Init); err != nil {
				return err
			}
		}
		idx := b.locals[n.Name]
		b.buf = append(b.buf, OpLocalSet)
		b.buf = AppendULEB128(b.buf, idx)
		return nil
	case *script.AssignmentExpression:
		if err := b.walk(n.Value); err != nil {
			return err
		}
		idx, ok := b.locals[n.Name]
		if !ok {
			return fmt.Errorf("unknown local %s", n.Name)
		}
		b.buf = append(b.buf, OpLocalSet)
		b.buf = AppendULEB128(b.buf, idx)
		return nil
	case *script.IfStatement:
		if err := b.walk(n.Condition); err != nil {
			return err
		}
		b.buf = append(b.buf, OpIf, BlockTypeEmpty)
		b.blockDepth++

		if err := b.walk(n.Body); err != nil {
			return err
		}
		if n.Else != nil {
			b.buf = append(b.buf, OpElse)
			if err := b.walk(n.Else); err != nil {
				return err
			}
		}
		b.buf = append(b.buf, OpEnd)
		b.blockDepth--

		return nil
	case *script.WhileStatement:
		b.buf = append(b.buf, OpBlock, BlockTypeEmpty)
		b.blockDepth++
		blockLabel := uint32(b.blockDepth)

		b.buf = append(b.buf, OpLoop, BlockTypeEmpty)
		b.blockDepth++
		loopLabel := uint32(b.blockDepth)

		b.loopStack = append(b.loopStack, loopEntry{
			blockLabel,
			loopLabel,
		})

		if err := b.walk(n.Condition); err != nil {
			return err
		}

		b.buf = append(b.buf, OpI32Eqz)
		b.buf = append(b.buf, OpBrIf)
		b.buf = AppendULEB128(b.buf, uint32(b.blockDepth)-blockLabel)

		if err := b.walk(n.Body); err != nil {
			return err
		}

		b.buf = append(b.buf, OpBr)

		b.buf = AppendULEB128(b.buf, uint32(b.blockDepth)-loopLabel)
		b.loopStack = b.loopStack[:len(b.loopStack)-1]
		b.blockDepth -= 2

		b.buf = append(b.buf, OpEnd, OpEnd)
		return nil
	case *script.BreakStatement:
		if len(b.loopStack) == 0 {
			return fmt.Errorf("break outside loop")
		}
		entry := b.loopStack[len(b.loopStack)-1]
		b.buf = append(b.buf, OpBr)
		b.buf = AppendULEB128(b.buf, uint32(b.blockDepth)-entry.blockLabel)
		return nil
	case *script.ContinueStatement:
		if len(b.loopStack) == 0 {
			return fmt.Errorf("continue outside loop")
		}
		entry := b.loopStack[len(b.loopStack)-1]
		b.buf = append(b.buf, OpBr)
		b.buf = AppendULEB128(b.buf, uint32(b.blockDepth)-entry.loopLabel)
		return nil
	case *script.FunctionCall:
		for _, arg := range n.Args {
			if err := b.walk(arg); err != nil {
				return err
			}
		}
		switch callee := n.Callee.(type) {
		case *script.Identifier:
			idx, ok := b.funcIndices[callee.Value]
			if !ok {
				return fmt.Errorf("unknown function %s", callee.Value)
			}

			b.buf = append(b.buf, OpCall)
			b.buf = AppendULEB128(b.buf, idx)
			return nil
		default:
			return fmt.Errorf("unsupported callee type %T", n.Callee)
		}
	case *script.UnaryExpression:
		if n.Operator == script.TokenType_Plus {
			return b.walk(n.Operand)
		}

		if n.Operator != script.TokenType_Minus {
			return fmt.Errorf("unsupported unary operator %v", n.Operator)
		}

		t := n.Operand.GetType()
		if t == nil {
			return fmt.Errorf("no type on unary operand")
		}

		switch t.Kind {
		case script.TypeKind_F64, script.TypeKind_Float:
			if err := b.walk(n.Operand); err != nil {
				return err
			}
			b.buf = append(b.buf, OpF64NEg)
			return nil
		case script.TypeKind_I64, script.TypeKind_Int:
			b.buf = append(b.buf, OpI64Const)
			b.buf = AppendULEB128(b.buf, 0)
			if err := b.walk(n.Operand); err != nil {
				return err
			}
			b.buf = append(b.buf, OpI64Sub)
			return nil
		default:
			return fmt.Errorf("unary minus not supported for type %s", t)
		}

	default:
		return fmt.Errorf("unhandled AST node %T", n)
	}
}

func (b *bodyEncoder) collectLocals(node script.AstNode) {
	switch n := node.(type) {
	case *script.Block:
		for _, s := range n.Stmts {
			b.collectLocals(s)
		}
	case *script.VariableDeclaration:
		b.locals[n.Name] = uint32(len(b.locals))
		t := n.GetType()
		if t != nil {
			b.localTypes = append(b.localTypes, valType(t))
		} else {
			b.localTypes = append(b.localTypes, ValI64)
		}
	case *script.WhileStatement:
		b.collectLocals(n.Body)
	case *script.IfStatement:
		b.collectLocals(n.Body)
		if n.Else != nil {
			b.collectLocals(n.Else)
		}
	}
}

func (b *bodyEncoder) walkShortCircuit(n *script.BinaryExpression) error {
	if err := b.walk(n.Left); err != nil {
		return err
	}
	b.buf = append(b.buf, OpIf, ValI32)
	b.blockDepth++

	if n.Operator == script.TokenType_AND {
		// a && b  →  if a then b else 0
		if err := b.walk(n.Right); err != nil {
			return err
		}
		b.buf = append(b.buf, OpElse)
		b.buf = append(b.buf, OpI32Const, 0)
	} else {
		// a || b  →  if a then 1 else b
		b.buf = append(b.buf, OpI32Const, 1)
		b.buf = append(b.buf, OpElse)
		if err := b.walk(n.Right); err != nil {
			return err
		}
	}

	b.buf = append(b.buf, OpEnd)
	b.blockDepth--
	return nil
}
