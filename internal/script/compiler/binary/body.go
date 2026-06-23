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

	strings *stringTable
}

func newBodyEncoder(f *script.FunctionDeclaration, sig funcSig, funcIndices map[string]uint32, strings *stringTable) *bodyEncoder {
	b := &bodyEncoder{locals: make(map[string]uint32), funcIndices: funcIndices, strings: strings}
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
		case *script.MemberAccess:
			objExpr, ok := callee.Object.(script.Expression)
			if !ok {
				return fmt.Errorf("number access object has no type")
			}

			objType := objExpr.GetType()
			if objType == nil {
				return fmt.Errorf("member access object type not set")
			}

			switch objType.Kind {
			case script.TypeKind_String:
				switch callee.Field {
				case "len":
					if err := b.walk(callee.Object); err != nil {
						return err
					}
					b.buf = append(b.buf, OpI32Load)
					b.buf = AppendULEB128(b.buf, 2)
					b.buf = AppendULEB128(b.buf, 0)

					b.buf = append(b.buf, OpI64ExtendI32S)
					return nil
				default:
					return fmt.Errorf("string has no method %s", callee.Field)
				}
			case script.TypeKind_Array:
				switch callee.Field {
				case "len":
					if err := b.walk(callee.Object); err != nil {
						return err
					}

					b.buf = append(b.buf, OpI32Load)
					b.buf = AppendULEB128(b.buf, 2) // Align
					b.buf = AppendULEB128(b.buf, 0) // Offset
					b.buf = append(b.buf, OpI64ExtendI32S)

					return nil
				default:
					return fmt.Errorf("array has no method %s", callee.Field)
				}
			default:
				return fmt.Errorf("method calls on %s not supported", objType)
			}

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
	case *script.StringLiteral:
		off, ok := b.strings.byValue[n.Value]
		if !ok {
			return fmt.Errorf("string %s not int table", n.Value)
		}

		b.buf = append(b.buf, OpI32Const)
		b.buf = AppendSLEB128(b.buf, int64(off))
		return nil
	case *script.ArrayLiteral:
		t := n.GetType()
		if t == nil || t.Element == nil {
			return fmt.Errorf("array literal missing element type")
		}

		elemSize := elemSizeOf(t.Element.Kind)
		storeOp, storeAlign := storeOpcode(t.Element.Kind)
		totalSize := uint32(4) + uint32(len(n.Elements))*elemSize

		arrTmp := b.addSyntheticLocal(ValI32)

		// bump: arrTmp = heapPtr; heapPtr += totalSize
		b.buf = append(b.buf, OpGlobalGet)
		b.buf = AppendULEB128(b.buf, 0)
		b.buf = append(b.buf, OpLocalTee)
		b.buf = AppendULEB128(b.buf, arrTmp)
		b.buf = append(b.buf, OpI32Const)
		b.buf = AppendSLEB128(b.buf, int64(totalSize))
		b.buf = append(b.buf, OpI32Add)
		b.buf = append(b.buf, OpGlobalSet)
		b.buf = AppendULEB128(b.buf, 0)
		// end allocator

		// store length at base+0
		b.buf = append(b.buf, OpLocalGet)
		b.buf = AppendULEB128(b.buf, arrTmp)
		b.buf = append(b.buf, OpI32Const)
		b.buf = AppendSLEB128(b.buf, int64(len(n.Elements)))
		b.buf = append(b.buf, OpI32Store)
		b.buf = AppendULEB128(b.buf, 2)
		b.buf = AppendULEB128(b.buf, 0)

		// store each element at base + 4 + i*elemSize
		for i, el := range n.Elements {
			offset := uint32(4) + uint32(i)*elemSize
			b.buf = append(b.buf, OpLocalGet)
			b.buf = AppendULEB128(b.buf, arrTmp)
			if err := b.walk(el); err != nil {
				return err
			}
			b.buf = append(b.buf, storeOp)
			b.buf = AppendULEB128(b.buf, storeAlign)
			b.buf = AppendULEB128(b.buf, offset)
		}

		// leave the base ptr on the stack for the parent
		b.buf = append(b.buf, OpLocalGet)
		b.buf = AppendULEB128(b.buf, arrTmp)
		return nil
	case *script.ArrayAccess:
		elemType := n.GetType()
		if elemType == nil {
			return fmt.Errorf("array access has no element type")
		}

		elemSize := elemSizeOf(elemType.Kind)
		loadOp, loadAlign := loadOpcode(elemType.Kind)
		if err := b.walk(n.Target); err != nil {
			return err
		}

		if err := b.walk(n.Index); err != nil {
			return err
		}

		b.buf = append(b.buf, OpI32WrapI64)

		b.buf = append(b.buf, OpI32Const)
		b.buf = AppendSLEB128(b.buf, int64(elemSize))
		b.buf = append(b.buf, OpI32Mul)

		b.buf = append(b.buf, OpI32Add)

		b.buf = append(b.buf, loadOp)
		b.buf = AppendULEB128(b.buf, loadAlign)
		b.buf = AppendULEB128(b.buf, 4)

		return nil
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

func (b *bodyEncoder) addSyntheticLocal(vt byte) uint32 {
	idx := uint32(len(b.locals))
	// The placeholder name keeps b.locals's len() advancing so the next
	// synthetic local gets a fresh index. Source code can't reference it
	// because plex identifiers don't start with "__".
	b.locals[fmt.Sprintf("__synth_%d", idx)] = idx
	b.localTypes = append(b.localTypes, vt)
	return idx
}
