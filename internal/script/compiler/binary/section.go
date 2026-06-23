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

func encodeCodeEntry(f *script.FunctionDeclaration, sig funcSig, funcIndices map[string]uint32, strs *stringTable, structs map[string]*structLayout) ([]byte, error) {
	var body []byte

	enc := newBodyEncoder(f, sig, funcIndices, strs, structs)

	// Walk first so any synthetic locals added during codegen (e.g. array
	// allocator temporaries) land in enc.localTypes before we write the
	// locals vec. The locals declaration must precede the body bytes in the
	// section, but it can't be finalised until the body is fully encoded.
	if err := enc.walk(f.Body); err != nil {
		return nil, err
	}

	body = AppendULEB128(body, uint32(len(enc.localTypes)))
	for _, vt := range enc.localTypes {
		body = AppendULEB128(body, 1)
		body = append(body, vt)
	}

	body = append(body, enc.buf...)
	body = append(body, OpUnreachable)
	body = append(body, OpEnd)

	var entry []byte
	entry = AppendULEB128(entry, uint32(len(body)))
	entry = append(entry, body...)

	return entry, nil
}

func encodeCodeSection(funcs []*script.FunctionDeclaration, sigs []funcSig, funcIndices map[string]uint32, strings *stringTable, structs map[string]*structLayout) ([]byte, error) {
	var body []byte
	body = AppendULEB128(body, uint32(len(funcs)))
	for i, f := range funcs {
		sig := sigs[i]

		entry, err := encodeCodeEntry(f, sig, funcIndices, strings, structs)
		if err != nil {
			return nil, err
		}
		body = append(body, entry...)
	}
	return section(SectionCode, body), nil
}

func encodeName(buf []byte, s string) []byte {
	buf = AppendULEB128(buf, uint32(len(s)))
	return append(buf, s...)
}

func encodeExportSection(exports []exportEntry) []byte {
	var body []byte
	body = AppendULEB128(body, uint32(len(exports)))
	for _, e := range exports {
		body = encodeName(body, e.name)
		body = append(body, e.kind)
		body = AppendULEB128(body, e.idx)
	}
	return section(SectionExport, body)
}

func encodeMemorySection() []byte {
	var body []byte
	body = AppendULEB128(body, 1)
	body = append(body, 0x00)
	body = AppendULEB128(body, 1)
	return section(SectionMemory, body)
}

func encodeDataSection(st *stringTable) []byte {
	var body []byte
	body = AppendULEB128(body, uint32(len(st.entries)))
	for _, e := range st.entries {
		body = AppendULEB128(body, 0)
		body = append(body, OpI32Const)
		body = AppendSLEB128(body, int64(e.offset))
		body = append(body, OpEnd)

		n := len(e.value)
		bytes := []byte{byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)}
		bytes = append(bytes, e.value...)

		body = AppendULEB128(body, uint32(len(bytes)))
		body = append(body, bytes...)
	}

	return section(SectionData, body)
}

func encodeGlobalSection(heapStart uint32) []byte {
	var body []byte

	body = AppendULEB128(body, 1)
	body = append(body, ValI32)
	body = append(body, 0x01)

	body = append(body, OpI32Const)
	body = AppendSLEB128(body, int64(heapStart))
	body = append(body, OpEnd)

	return section(SectionGlobal, body)
}
