package binary_wasm

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"

	"github.com/VisualSource/plex/internal/script"
)

type bodyEncoder struct {
	buf []byte
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
			binary.LittleEndian.AppendUint64(raw[:], math.Float64bits(v))
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
	default:
		return fmt.Errorf("unhandled AST node %T", n)
	}
}
