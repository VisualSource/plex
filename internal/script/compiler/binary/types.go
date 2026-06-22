package binary_wasm

import "github.com/VisualSource/plex/internal/script"

const (
	ValI32 byte = 0x7F
	ValI64 byte = 0x7E
	ValF32 byte = 0x7D
	ValF64 byte = 0x7C
)

func valType(t *script.Type) byte {
	switch t.Kind {
	case script.TypeKind_Int, script.TypeKind_I64:
		return ValI64
	case script.TypeKind_Float, script.TypeKind_F64:
		return ValF64
	case script.TypeKind_F32:
		return ValF32
	default:
		return ValI32
	}
}
