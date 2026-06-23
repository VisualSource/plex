package binary_wasm

import (
	"fmt"

	"github.com/VisualSource/plex/internal/script"
)

const (
	OpUnreachable byte = 0x00
	OpBlock       byte = 0x02
	OpLoop        byte = 0x03
	OpIf          byte = 0x04
	OpElse        byte = 0x05

	OpCall byte = 0x10
	OpEnd  byte = 0x0B
	OpBr   byte = 0x0C
	OpBrIf byte = 0x0D

	OpReturn    byte = 0x0F
	OpGlobalGet byte = 0x23
	OpGlobalSet byte = 0x24

	OpDrop byte = 0x1A

	OpLocalGet byte = 0x20
	OpLocalSet byte = 0x21
	OpLocalTee byte = 0x22
	OpI32Load  byte = 0x28
	OpI64Load  byte = 0x29

	OpF64Load byte = 0x2B

	OpI32Store byte = 0x36
	OpI64Store byte = 0x37

	OpF64Store byte = 0x39

	BlockTypeEmpty byte = 0x40

	OpI32Const byte = 0x41
	OpI64Const byte = 0x42
	OpF64Const byte = 0x44

	OpI32Eqz byte = 0x45
	OpI32Eq  byte = 0x46
	OpI32Ne  byte = 0x47
	OpI32LtS byte = 0x48

	OpI32GtS byte = 0x4A
	OpI32LeS byte = 0x4C
	OpI32GeS byte = 0x4E
	OpI32GeU      = 0x4F

	OpI64Eq  byte = 0x51
	OpI64Ne  byte = 0x52
	OpI64LtS byte = 0x53
	OpI64GtS byte = 0x55
	OpI64LeS byte = 0x57
	OpI64GeS byte = 0x59

	OpF64Eq byte = 0x61
	OpF64Ne byte = 0x62
	OpF64Lt byte = 0x63
	OpF64Gt byte = 0x64
	OpF64Le byte = 0x65
	OpF64Ge byte = 0x66

	OpI32Add  byte = 0x6A
	OpI32Sub  byte = 0x6B
	OpI32Mul  byte = 0x6C
	OpI32DivS byte = 0x6D
	OpI32RemS byte = 0x6F

	OpI32And byte = 0x71
	OpI32Or  byte = 0x72

	OpI64Add  byte = 0x7C
	OpI64Sub  byte = 0x7D
	OpI64Mul  byte = 0x7E
	OpI64DivS byte = 0x7F
	OpI64RemS byte = 0x81

	OpF64NEg byte = 0x9A

	OpF64Add byte = 0xA0
	OpF64Sub byte = 0xA1
	OpF64Mul byte = 0xA2
	OpF64Div byte = 0xA3

	OpI32WrapI64 byte = 0xA7

	OpI64ExtendI32S byte = 0xAC
)

const (
	ExportFunc   byte = 0x00
	ExportTable  byte = 0x01
	ExportMemory byte = 0x02
	ExportGlobal byte = 0x03
)

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

func cmpOpcode(k script.TypeKind, op script.TokenType) (byte, error) {
	switch k {
	case script.TypeKind_I64, script.TypeKind_Int:
		switch op {
		case script.TokenType_EqualEqual:
			return OpI64Eq, nil
		case script.TokenType_NotEqual:
			return OpI64Ne, nil
		case script.TokenType_LessThen:
			return OpI64LtS, nil
		case script.TokenType_GreaterThen:
			return OpI64GtS, nil
		case script.TokenType_LessThenOrEqual:
			return OpI64LeS, nil
		case script.TokenType_GreaterThenOrEqaul:
			return OpI64GeS, nil

		}
	case script.TypeKind_I32:
		switch op {
		case script.TokenType_EqualEqual:
			return OpI32Eq, nil
		case script.TokenType_NotEqual:
			return OpI32Ne, nil
		case script.TokenType_LessThen:
			return OpI32LtS, nil
		case script.TokenType_GreaterThen:
			return OpI32GtS, nil
		case script.TokenType_LessThenOrEqual:
			return OpI32LeS, nil
		case script.TokenType_GreaterThenOrEqaul:
			return OpI32GeS, nil
		}
	case script.TypeKind_Bool:
		switch op {
		case script.TokenType_AND:
			return OpI32And, nil
		case script.TokenType_OR:
			return OpI32Or, nil
		}
	case script.TypeKind_F64, script.TypeKind_Float:
		switch op {
		case script.TokenType_EqualEqual:
			return OpF64Eq, nil
		case script.TokenType_NotEqual:
			return OpF64Ne, nil
		case script.TokenType_LessThen:
			return OpF64Lt, nil
		case script.TokenType_GreaterThen:
			return OpF64Gt, nil
		case script.TokenType_LessThenOrEqual:
			return OpF64Le, nil
		case script.TokenType_GreaterThenOrEqaul:
			return OpF64Ge, nil
		}
	}
	return 0, fmt.Errorf("no comparison opcode for %s %s", k, op)
}

func isCmpOp(op script.TokenType) bool {
	switch op {
	case script.TokenType_EqualEqual,
		script.TokenType_NotEqual,
		script.TokenType_LessThen,
		script.TokenType_GreaterThen,
		script.TokenType_LessThenOrEqual,
		script.TokenType_GreaterThenOrEqaul,
		script.TokenType_AND,
		script.TokenType_OR:
		return true

	default:
		return false
	}
}

func storeOpcode(k script.TypeKind) (op byte, align uint32) {
	switch k {
	case script.TypeKind_I32:
		return OpI32Store, 2
	case script.TypeKind_I64, script.TypeKind_Int:
		return OpI64Store, 3
	case script.TypeKind_F64, script.TypeKind_Float:
		return OpF64Store, 3
	default:
		return OpI32Store, 2 // pointer-as-i32
	}
}

func loadOpcode(k script.TypeKind) (op byte, align uint32) {
	switch k {
	case script.TypeKind_I32:
		return OpI32Load, 2
	case script.TypeKind_I64, script.TypeKind_Int:
		return OpI64Load, 3
	case script.TypeKind_F64, script.TypeKind_Float:
		return OpF64Load, 3
	default:
		return OpI32Load, 2 // pointer-as-i32
	}
}
