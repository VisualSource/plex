package binary_wasm

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"

	"github.com/VisualSource/plex/internal/script"
)

type bodyEncoder struct {
	buf    []byte
	locals map[string]uint32
}

func newBodyEncoder(f *script.FunctionDeclaration, sig funcSig) *bodyEncoder {
	b := &bodyEncoder{locals: make(map[string]uint32)}
	var idx uint32
	if sig.isMethod {
		b.locals["self"] = idx
		idx++
	}
	for _, p := range f.Params {
		b.locals[p.Name] = idx
		idx++
	}

	return b
}

func (b *bodyEncoder) walk(node script.AstNode) error {
	switch n := node.(type) {
	case *script.Block:
		for _, s := range n.Stmts {
			if err := b.walk(s); err != nil {
				return err
			}
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
		if err := b.walk(n.Left); err != nil {
			return err
		}
		if err := b.walk(n.Right); err != nil {
			return err
		}
		t := n.GetType()
		if t == nil {
			return fmt.Errorf("no type on binary expression")
		}
		op, err := arithOpcode(t.Kind, n.Operator)
		if err != nil {
			return err
		}
		b.buf = append(b.buf, op)
		return nil
	default:
		return fmt.Errorf("unhandled AST node %T", n)
	}
}

func arithOpcode(k script.TypeKind, op script.TokenType) (byte, error) {
	switch k {
	case script.TypeKind_I32:
		switch op {
		case script.TokenType_Plus:
			return OpI32Add, nil
		case script.TokenType_Minus:
			return OpI32Sub, nil
		case script.TokenType_Star:
			return OpI32Mul, nil
		case script.TokenType_Div:
			return OpI32DivS, nil
		case script.TokenType_Mod:
			return OpI32RemS, nil
		}
	case script.TypeKind_I64, script.TypeKind_Int:
		switch op {
		case script.TokenType_Plus:
			return OpI64Add, nil
		case script.TokenType_Minus:
			return OpI64Sub, nil
		case script.TokenType_Star:
			return OpI64Mul, nil
		case script.TokenType_Div:
			return OpI64DivS, nil
		case script.TokenType_Mod:
			return OpI64RemS, nil
		}
	case script.TypeKind_Float, script.TypeKind_F64:
		switch op {
		case script.TokenType_Plus:
			return OpF64Add, nil
		case script.TokenType_Minus:
			return OpF64Sub, nil
		case script.TokenType_Star:
			return OpF64Mul, nil
		case script.TokenType_Div:
			return OpF64Div, nil
		}
	}
	return 0, fmt.Errorf("no opcode for %s %v", k, op)
}
