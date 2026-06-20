package script

import (
	"fmt"
)

type TypeKind uint

const (
	TypeKind_I32 TypeKind = iota
	TypeKind_I64
	TypeKind_F32
	TypeKind_F64
	TypeKind_Int
	TypeKind_Float
	TypeKind_String
	TypeKind_Array
	TypeKind_Struct
	TypeKind_Void
	TypeKind_Unknown
)

func (t TypeKind) String() string {
	switch t {
	case TypeKind_I32:
		return "i32"
	case TypeKind_I64:
		return "i64"
	case TypeKind_F32:
		return "f32"
	case TypeKind_F64:
		return "f64"
	case TypeKind_Int:
		return "int"
	case TypeKind_Float:
		return "float"
	case TypeKind_String:
		return "string"
	case TypeKind_Array:
		return "array"
	case TypeKind_Struct:
		return "struct"
	case TypeKind_Void:
		return "void"
	case TypeKind_Unknown:
		return "unknown"
	default:
		return fmt.Sprintf("TypeKind(%d)", t)
	}
}

type Type struct {
	Kind     TypeKind
	Element  *Type
	Struct   string
	Nullable bool
}

func (t *Type) String() string {
	switch t.Kind {
	case TypeKind_Array:
		null := ""
		if t.Nullable {
			null = "?"
		}
		return fmt.Sprintf("%s[]%s", t.Element, null)
	case TypeKind_Struct:
		null := ""
		if t.Nullable {
			null = "?"
		}
		return fmt.Sprintf("%s%s", t.Struct, null)
	default:
		null := ""
		if t.Nullable {
			null = "?"
		}
		return fmt.Sprintf("%s%s", t.Kind, null)
	}
}
