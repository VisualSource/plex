package compiler

import "github.com/VisualSource/plex/internal/script"

func getWasmType(tp *script.Type) string {
	switch tp.Kind {
	case script.TypeKind_Int, script.TypeKind_I64:
		return "i64"
	case script.TypeKind_Float, script.TypeKind_F64:
		return "f64"
	case script.TypeKind_F32:
		return "f32"
	case script.TypeKind_Array, script.TypeKind_Struct, script.TypeKind_String:
		return "i32"
	default:
		return "i32"
	}
}

func sizeOf(value string) int {
	switch value {
	case "f64", "i64":
		return 8
	case "f32", "i32":
		return 4
	default:
		return 8
	}
}
