package binary_wasm

const (
	OpEnd      byte = 0x0B
	OpReturn   byte = 0x0F
	OpLocalGet byte = 0x20
	OpI32Const byte = 0x41
	OpI64Const byte = 0x42
	OpF64Const byte = 0x44
)

const (
	ExportFunc   byte = 0x00
	ExportTable  byte = 0x01
	ExportMemory byte = 0x02
	ExportGlobal byte = 0x03
)
