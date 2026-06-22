package binary_wasm

import "github.com/VisualSource/plex/internal/script"

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

func encodeFunctionSection(sigs []funcSig) []byte {
	var body []byte
	body = AppendULEB128(body, uint32(len(sigs)))
	for i := range sigs {
		body = AppendULEB128(body, uint32(i))
	}
	return section(SectionFunction, body)
}

func encodeCodeEntry(f *script.FunctionDeclaration) ([]byte, error) {
	var body []byte

	body = AppendULEB128(body, 0)

	enc := &bodyEncoder{}
	if err := enc.walk(f.Body); err != nil {
		return nil, err
	}

	body = append(body, enc.buf...)
	body = append(body, OpEnd)

	var entry []byte
	entry = AppendULEB128(entry, uint32(len(body)))
	entry = append(entry, body...)

	return entry, nil
}

func encodeCodeSection(funcs []*script.FunctionDeclaration) ([]byte, error) {
	var body []byte
	body = AppendULEB128(body, uint32(len(funcs)))
	for _, f := range funcs {
		entry, err := encodeCodeEntry(f)
		if err != nil {
			return nil, err
		}
		body = append(body, entry...)
	}
	return section(SectionCode, body), nil
}
