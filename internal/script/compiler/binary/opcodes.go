package binary_wasm

const (
	OpEnd      byte = 0x0B
	OpReturn   byte = 0x0F
	OpLocalGet byte = 0x20
	OpI32Const byte = 0x41
	OpI64Const byte = 0x42
	OpF64Const byte = 0x44

	OpI32Add  byte = 0x6A
	OpI32Sub  byte = 0x6B
	OpI32Mul  byte = 0x6C
	OpI32DivS byte = 0x6D
	OpI32RemS byte = 0x6F

	OpI64Add  byte = 0x7C
	OpI64Sub  byte = 0x7D
	OpI64Mul  byte = 0x7F
	OpI64DivS byte = 0x7F
	OpI64RemS byte = 0x81

	OpF64Add byte = 0xA0
	OpF64Sub byte = 0xA1
	OpF64Mul byte = 0xA2
	OpF64Div byte = 0xA3
)

const (
	ExportFunc   byte = 0x00
	ExportTable  byte = 0x01
	ExportMemory byte = 0x02
	ExportGlobal byte = 0x03
)
