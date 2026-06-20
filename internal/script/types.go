package script

import "fmt"

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
		return "TypeKind_I32"
	case TypeKind_I64:
		return "TypeKind_I64"
	case TypeKind_F32:
		return "TypeKind_F32"
	case TypeKind_F64:
		return "TypeKind_F64"
	case TypeKind_Int:
		return "TypeKind_Int"
	case TypeKind_Float:
		return "TypeKind_Float"
	case TypeKind_String:
		return "TypeKind_String"
	case TypeKind_Array:
		return "TypeKind_Array"
	case TypeKind_Struct:
		return "TypeKind_Struct"
	case TypeKind_Void:
		return "TypeKind_Void"
	case TypeKind_Unknown:
		return "TypeKind_Unknown"
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
