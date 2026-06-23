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

	structs map[string]*structLayout
}

func newBodyEncoder(f *script.FunctionDeclaration, sig funcSig, funcIndices map[string]uint32, strings *stringTable, structs map[string]*structLayout) *bodyEncoder {
	b := &bodyEncoder{locals: make(map[string]uint32), funcIndices: funcIndices, strings: strings, structs: structs}
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
		switch callee := n.Callee.(type) {
		case *script.Identifier:
			// Check for struct constructor first — walkStructConstructor
			// handles its own arg walking. If we fall through to the generic
			// arg-walk below, we must not walk args twice.
			if layout, ok := b.structs[callee.Value]; ok {
				return b.walkStructConstructor(layout, n.Args)
			}

			for _, arg := range n.Args {
				if err := b.walk(arg); err != nil {
					return err
				}
			}

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
					if len(n.Args) != 0 {
						return fmt.Errorf("len method was expecting no args but was given %d", len(n.Args))
					}
					if err := b.walk(callee.Object); err != nil {
						return err
					}

					b.buf = append(b.buf, OpI32Load)
					b.buf = AppendULEB128(b.buf, 2) // Align
					b.buf = AppendULEB128(b.buf, 0) // Offset
					b.buf = append(b.buf, OpI64ExtendI32S)

					return nil
				case "append":
					if len(n.Args) != 1 {
						return fmt.Errorf("append method was expecting 1 arg but was given %d", len(n.Args))
					}

					arr := b.addSyntheticLocal(ValI32)
					if err := b.walk(callee.Object); err != nil {
						return err
					}
					b.buf = append(b.buf, OpLocalTee)
					b.buf = AppendULEB128(b.buf, arr)

					// compute write_addr = ptr + 4 + len*elemSize
					eSize := elemSizeOf(objType.Element.Kind)
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, arr)
					b.buf = append(b.buf, OpI32Load)
					b.buf = AppendULEB128(b.buf, 2)
					b.buf = AppendULEB128(b.buf, 0)
					b.buf = append(b.buf, OpI32Const)
					b.buf = AppendULEB128(b.buf, eSize)
					b.buf = append(b.buf, OpI32Mul)
					b.buf = append(b.buf, OpI32Const, 4)
					b.buf = append(b.buf, OpI32Add)
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, arr)
					b.buf = append(b.buf, OpI32Add)

					// store the new element
					if err := b.walk(n.Args[0]); err != nil {
						return err
					}
					stOp, stAlign := storeOpcode(objType.Element.Kind)
					b.buf = append(b.buf, stOp)
					b.buf = AppendULEB128(b.buf, uint32(stAlign))
					b.buf = AppendULEB128(b.buf, 0)

					// increment length at ptr
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, arr)
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, arr)
					b.buf = append(b.buf, OpI32Load)
					b.buf = AppendULEB128(b.buf, 2)
					b.buf = AppendULEB128(b.buf, 0)
					b.buf = append(b.buf, OpI32Const, 1)
					b.buf = append(b.buf, OpI32Add)
					b.buf = append(b.buf, OpI32Store)
					b.buf = AppendULEB128(b.buf, 2)
					b.buf = AppendULEB128(b.buf, 0)
					return nil
				case "remove":
					if len(n.Args) != 1 {
						return fmt.Errorf("remove method was expecting 1 arg but was given %d", len(n.Args))
					}
					eKind := objType.Element.Kind
					eSize := elemSizeOf(eKind)
					loadOp, loadAlign := loadOpcode(eKind)
					stOp, stAlign := storeOpcode(eKind)

					arrLocal := b.addSyntheticLocal(ValI32)
					idxLocal := b.addSyntheticLocal(ValI32)
					retLocal := b.addSyntheticLocal(valType(objType.Element))
					lenLocal := b.addSyntheticLocal(ValI32)
					curLocal := b.addSyntheticLocal(ValI32)

					// save arr
					if err := b.walk(callee.Object); err != nil {
						return err
					}
					b.buf = append(b.buf, OpLocalSet)
					b.buf = AppendULEB128(b.buf, arrLocal)

					// save idx (int arrives as i64 → unwrap to i32)
					if err := b.walk(n.Args[0]); err != nil {
						return err
					}
					b.buf = append(b.buf, OpI32WrapI64, OpLocalSet)
					b.buf = AppendULEB128(b.buf, idxLocal)

					// save elem to return: arr + 4 + idx*eSize
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, arrLocal)
					b.buf = append(b.buf, OpI32Const, 4, OpI32Add)
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, idxLocal)
					b.buf = append(b.buf, OpI32Const)
					b.buf = AppendULEB128(b.buf, eSize)
					b.buf = append(b.buf, OpI32Mul, OpI32Add)
					b.buf = append(b.buf, loadOp)
					b.buf = AppendULEB128(b.buf, loadAlign)
					b.buf = AppendULEB128(b.buf, 0)
					b.buf = append(b.buf, OpLocalSet)
					b.buf = AppendULEB128(b.buf, retLocal)

					// new_len = old_len - 1
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, arrLocal)
					b.buf = append(b.buf, OpI32Load)
					b.buf = AppendULEB128(b.buf, 2)
					b.buf = AppendULEB128(b.buf, 0)
					b.buf = append(b.buf, OpI32Const, 1, OpI32Sub)
					b.buf = append(b.buf, OpLocalSet)
					b.buf = AppendULEB128(b.buf, lenLocal)

					// cur = idx
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, idxLocal)
					b.buf = append(b.buf, OpLocalSet)
					b.buf = AppendULEB128(b.buf, curLocal)

					// internal shift loop — NOT on loopStack
					b.buf = append(b.buf, OpBlock, BlockTypeEmpty)
					b.blockDepth++
					blockMark := uint32(b.blockDepth)
					b.buf = append(b.buf, OpLoop, BlockTypeEmpty)
					b.blockDepth++
					loopMark := uint32(b.blockDepth)

					// exit: cur >= len → br_if out of block
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, curLocal)
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, lenLocal)
					b.buf = append(b.buf, OpI32GeS, OpBrIf)
					b.buf = AppendULEB128(b.buf, uint32(b.blockDepth)-blockMark)

					// dst: arr + 4 + cur*eSize
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, arrLocal)
					b.buf = append(b.buf, OpI32Const, 4, OpI32Add)
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, curLocal)
					b.buf = append(b.buf, OpI32Const)
					b.buf = AppendULEB128(b.buf, eSize)
					b.buf = append(b.buf, OpI32Mul, OpI32Add)

					// src: arr + 4 + (cur+1)*eSize
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, arrLocal)
					b.buf = append(b.buf, OpI32Const, 4, OpI32Add)
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, curLocal)
					b.buf = append(b.buf, OpI32Const, 1, OpI32Add)
					b.buf = append(b.buf, OpI32Const)
					b.buf = AppendULEB128(b.buf, eSize)
					b.buf = append(b.buf, OpI32Mul, OpI32Add)
					b.buf = append(b.buf, loadOp)
					b.buf = AppendULEB128(b.buf, loadAlign)
					b.buf = AppendULEB128(b.buf, 0)

					// store dst ← src
					b.buf = append(b.buf, stOp)
					b.buf = AppendULEB128(b.buf, uint32(stAlign))
					b.buf = AppendULEB128(b.buf, 0)

					// cur++
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, curLocal)
					b.buf = append(b.buf, OpI32Const, 1, OpI32Add)
					b.buf = append(b.buf, OpLocalSet)
					b.buf = AppendULEB128(b.buf, curLocal)

					// continue loop
					b.buf = append(b.buf, OpBr)
					b.buf = AppendULEB128(b.buf, uint32(b.blockDepth)-loopMark)

					b.blockDepth -= 2
					b.buf = append(b.buf, OpEnd, OpEnd)

					// write new length
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, arrLocal)
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, lenLocal)
					b.buf = append(b.buf, OpI32Store)
					b.buf = AppendULEB128(b.buf, 2)
					b.buf = AppendULEB128(b.buf, 0)

					// return removed element
					b.buf = append(b.buf, OpLocalGet)
					b.buf = AppendULEB128(b.buf, retLocal)
					return nil
				default:
					return fmt.Errorf("array has no method %s", callee.Field)
				}
			case script.TypeKind_Struct:
				// Method dispatch: push receiver, then args, then call __Struct__method.
				if err := b.walk(callee.Object); err != nil {
					return err
				}
				for _, arg := range n.Args {
					if err := b.walk(arg); err != nil {
						return err
					}
				}
				mangled := fmt.Sprintf("__%s__%s", objType.Struct, callee.Field)
				idx, ok := b.funcIndices[mangled]
				if !ok {
					return fmt.Errorf("struct %s has no method %s", objType.Struct, callee.Field)
				}
				b.buf = append(b.buf, OpCall)
				b.buf = AppendULEB128(b.buf, idx)
				return nil
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

		arrBase := b.addSyntheticLocal(ValI32)
		arrIdx := b.addSyntheticLocal(ValI32)

		if err := b.walk(n.Target); err != nil {
			return err
		}

		b.buf = append(b.buf, OpLocalSet)
		b.buf = AppendULEB128(b.buf, arrBase)

		if err := b.walk(n.Index); err != nil {
			return err
		}

		b.buf = append(b.buf, OpI32WrapI64)

		b.buf = append(b.buf, OpLocalSet)
		b.buf = AppendULEB128(b.buf, arrIdx)

		b.buf = append(b.buf, OpLocalGet)
		b.buf = AppendULEB128(b.buf, arrIdx)

		b.buf = append(b.buf, OpLocalGet)
		b.buf = AppendULEB128(b.buf, arrBase)

		b.buf = append(b.buf, OpI32Load)
		b.buf = AppendULEB128(b.buf, 2)
		b.buf = AppendULEB128(b.buf, 0)

		b.buf = append(b.buf, OpI32GeU)

		b.buf = append(b.buf, OpIf, BlockTypeEmpty)

		b.blockDepth++
		b.buf = append(b.buf, OpUnreachable)
		b.buf = append(b.buf, OpEnd)
		b.blockDepth--

		elemSize := elemSizeOf(elemType.Kind)
		loadOp, loadAlign := loadOpcode(elemType.Kind)

		b.buf = append(b.buf, OpLocalGet)
		b.buf = AppendULEB128(b.buf, arrBase)

		b.buf = append(b.buf, OpLocalGet)
		b.buf = AppendULEB128(b.buf, arrIdx)

		b.buf = append(b.buf, OpI32Const)
		b.buf = AppendSLEB128(b.buf, int64(elemSize))

		b.buf = append(b.buf, OpI32Mul)
		b.buf = append(b.buf, OpI32Add)

		b.buf = append(b.buf, loadOp)
		b.buf = AppendULEB128(b.buf, loadAlign)
		b.buf = AppendULEB128(b.buf, 4)

		return nil
	case *script.ArrayAssignment:
		elemType := n.Target.GetType()
		if elemType == nil {
			return fmt.Errorf("array assignment target has no element type")
		}
		elemSize := elemSizeOf(elemType.Kind)
		storeOp, storeAlign := storeOpcode(elemType.Kind)

		arrBase := b.addSyntheticLocal(ValI32)
		arrIdx := b.addSyntheticLocal(ValI32)

		if err := b.walk(n.Target.Target); err != nil {
			return err
		}

		b.buf = append(b.buf, OpLocalSet)
		b.buf = AppendULEB128(b.buf, arrBase)

		if err := b.walk(n.Target.Index); err != nil {
			return err
		}

		b.buf = append(b.buf, OpI32WrapI64)

		b.buf = append(b.buf, OpLocalSet)
		b.buf = AppendULEB128(b.buf, arrIdx)

		// start bounds check
		b.buf = append(b.buf, OpLocalGet)
		b.buf = AppendULEB128(b.buf, arrIdx)

		b.buf = append(b.buf, OpLocalGet)
		b.buf = AppendULEB128(b.buf, arrBase)

		b.buf = append(b.buf, OpI32Load)
		b.buf = AppendULEB128(b.buf, 2)
		b.buf = AppendULEB128(b.buf, 0)
		b.buf = append(b.buf, OpI32GeU)

		b.buf = append(b.buf, OpIf, BlockTypeEmpty)
		b.blockDepth++
		b.buf = append(b.buf, OpUnreachable)
		b.buf = append(b.buf, OpEnd)
		b.blockDepth--

		// end bounds check

		b.buf = append(b.buf, OpLocalGet)
		b.buf = AppendULEB128(b.buf, arrBase)
		b.buf = append(b.buf, OpLocalGet)
		b.buf = AppendULEB128(b.buf, arrIdx)

		b.buf = append(b.buf, OpI32Const)
		b.buf = AppendSLEB128(b.buf, int64(elemSize))
		b.buf = append(b.buf, OpI32Mul)
		b.buf = append(b.buf, OpI32Add)

		if err := b.walk(n.Value); err != nil {
			return err
		}

		b.buf = append(b.buf, storeOp)
		b.buf = AppendULEB128(b.buf, storeAlign)
		b.buf = AppendULEB128(b.buf, 4)
		return nil
	case *script.MemberAccess:
		objExpr, ok := n.Object.(script.Expression)
		if !ok {
			return fmt.Errorf("member access object has no type")
		}
		objType := objExpr.GetType()
		if objType == nil || objType.Kind != script.TypeKind_Struct {
			return fmt.Errorf("member access on non-struct type")
		}
		layout, ok := b.structs[objType.Struct]
		if !ok {
			return fmt.Errorf("unknown struct %s", objType.Struct)
		}
		field, ok := layout.findField(n.Field)
		if !ok {
			return fmt.Errorf("struct %s has no field %s", layout.name, n.Field)
		}
		loadOp, loadAlign := loadOpcode(field.kind)

		if err := b.walk(n.Object); err != nil {
			return err
		}
		b.buf = append(b.buf, loadOp)
		b.buf = AppendULEB128(b.buf, loadAlign)
		b.buf = AppendULEB128(b.buf, field.offset)
		return nil
	case *script.MemberAssignment:
		objExpr, ok := n.Object.(script.Expression)
		if !ok {
			return fmt.Errorf("member assignment object has no type")
		}
		objType := objExpr.GetType()
		if objType == nil || objType.Kind != script.TypeKind_Struct {
			return fmt.Errorf("member assignment on non-struct type")
		}
		layout, _ := b.structs[objType.Struct]
		field, ok := layout.findField(n.Field)
		if !ok {
			return fmt.Errorf("struct %s has no field %s", layout.name, n.Field)
		}
		storeOp, storeAlign := storeOpcode(field.kind)

		if err := b.walk(n.Object); err != nil {
			return err
		}
		if err := b.walk(n.Value); err != nil {
			return err
		}
		b.buf = append(b.buf, storeOp)
		b.buf = AppendULEB128(b.buf, storeAlign)
		b.buf = AppendULEB128(b.buf, field.offset)

		return nil
	case *script.TernaryExpression:
		t := n.GetType()
		if t == nil {
			return fmt.Errorf("ternary: no type")
		}

		if err := b.walk(n.Condition); err != nil {
			return err
		}
		b.buf = append(b.buf, OpIf, valType(t))
		if err := b.walk(n.TrueBlock); err != nil {
			return err
		}
		b.buf = append(b.buf, OpElse)
		if err := b.walk(n.FalseBlock); err != nil {
			return err
		}
		b.buf = append(b.buf, OpEnd)
		return nil
	case *script.CastExpression:
		if err := b.walk(n.Expr); err != nil {
			return err
		}
		from := n.Expr.(script.Expression).GetType()
		to := n.GetType()
		op, ok := castOpcode(from, to)

		if !ok {
			return fmt.Errorf("no cast from %s to %s", from, to)
		}

		if op != 0 {
			b.buf = append(b.buf, op)
		}
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

func (b *bodyEncoder) walkStructConstructor(layout *structLayout, args []script.AstNode) error {
	tmp := b.addSyntheticLocal(ValI32)

	// bump: tmp = heapPtr; heapPtr += layout.size
	b.buf = append(b.buf, OpGlobalGet)
	b.buf = AppendULEB128(b.buf, 0)
	b.buf = append(b.buf, OpLocalTee)
	b.buf = AppendULEB128(b.buf, tmp)
	b.buf = append(b.buf, OpI32Const)
	b.buf = AppendSLEB128(b.buf, int64(layout.size))
	b.buf = append(b.buf, OpI32Add)
	b.buf = append(b.buf, OpGlobalSet)
	b.buf = AppendULEB128(b.buf, 0)

	// end allocator

	// store each field at its layout offset
	for i, arg := range args {
		f := layout.fields[i]
		storeOp, storeAlign := storeOpcode(f.kind)

		b.buf = append(b.buf, OpLocalGet)
		b.buf = AppendULEB128(b.buf, tmp)
		if err := b.walk(arg); err != nil {
			return err
		}
		b.buf = append(b.buf, storeOp)
		b.buf = AppendULEB128(b.buf, storeAlign)
		b.buf = AppendULEB128(b.buf, f.offset)
	}

	// leave base pointer on the stack
	b.buf = append(b.buf, OpLocalGet)
	b.buf = AppendULEB128(b.buf, tmp)
	return nil
}
