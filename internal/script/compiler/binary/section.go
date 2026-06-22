package binary_wasm

const (
	SectionType     = 1
	SectionImport   = 2
	SectionFunction = 3
	SectionTable    = 4
	SectionMemory   = 5
	SectionGlobal   = 6
	SectionExport   = 7
	SectionStart    = 8
	SectionElement  = 9
	SectionCode     = 10
	SectionData     = 11
)

func section(id byte, body []byte) []byte {
	out := []byte{id}
	out = AppendULEB128(out, uint32(len(body)))
	return append(out, body...)
}

func encodeTypeSection(sigs []funcSig) []byte {
	var body []byte
	body = AppendULEB128(body, uint32(len(sigs)))
	for _, s := range sigs {
		body = append(body, 0x60)
		body = AppendULEB128(body, uint32(len(s.params)))
		body = append(body, s.params...)
		body = AppendULEB128(body, uint32(len(s.results)))
		body = append(body, s.results...)
	}

	return section(SectionType, body)
}
