package script

type TypeKind uint

const (
	TypeKind_I32 TypeKind = iota
	TypeKind_I64
	TypeKind_F32
	TypeKind_F64
	TypeKind_Int
	TypeKind_Uint
	TypeKind_Float
	TypeKind_Nil
	TypeKind_String
	TypeKind_Array
	TypeKind_Struct
	TypeKind_Void
	TypeKind_Unknown
)

type Type struct {
	Kind    TypeKind
	Element *Type
	Struct  string
}
