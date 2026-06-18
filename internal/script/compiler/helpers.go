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

func resolveType(node script.AstNode) (string, string) {
	// may need ref to compiter struct for type look up but should be fine for now.

	if t, ok := node.(*script.TypeExpr); ok {
		if !t.IsArray {
			switch t.Name {
			case "i64", "int":
				return "i64", ""
			case "f64", "float":
				return "f64", ""
			case "f32":
				return "f32", ""
			case "nil":
				// not sure what type nil should be yet,
				// but should point to a undefined/unset value
			case "i32", "string":
				// string is a pointer to starting offset
				return "i32", ""
			default:
				// unknown type, i32,string are i32 pointers
				// can only be structs right now
				// as creating named types for like i64 can't be done yet
				// so return i32 for a pointer
				return "i32", t.Name
			}
		} else {
			return "i32", "" // array type is a pointer to the start of the array
		}
	}

	return "f64", ""
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
