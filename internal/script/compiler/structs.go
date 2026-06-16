package compiler

type stringEntry struct {
	value  string
	offset int
}

type Local struct {
	Name  string
	Type  string
	Owner string
}

type structField struct {
	Name   string
	Type   string
	Offset int
}

type structDef struct {
	Name   string
	Fields []structField
	Size   int
}
