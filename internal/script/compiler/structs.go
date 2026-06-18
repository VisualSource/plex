package compiler

type WasmType string

const (
	Wasm_I64 WasmType = "i64"
	Wasm_F64 WasmType = "f64"
	Wasm_I32 WasmType = "i32"
	Wasm_F32 WasmType = "f32"
)

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
	Name    string
	Fields  []structField
	Methods map[string]string
	Size    int
}
