package binary_wasm

import "github.com/VisualSource/plex/internal/script"

type funcSig struct {
	name     string
	params   []byte
	results  []byte
	isMethod bool
}

type exportEntry struct {
	name string
	kind byte
	idx  uint32
}

type loopEntry struct {
	blockLabel uint32
	loopLabel  uint32
}

type stringEntry struct {
	value  string
	offset uint32
}

type stringTable struct {
	entries []stringEntry
	byValue map[string]uint32
	next    uint32
}

func (st *stringTable) add(v string) uint32 {
	if off, ok := st.byValue[v]; ok {
		return off
	}

	off := st.next
	st.entries = append(st.entries, stringEntry{v, off})
	if st.byValue == nil {
		st.byValue = make(map[string]uint32)
	}
	st.byValue[v] = off
	st.next += uint32(4 + len(v))
	return off
}

type structField struct {
	name   string
	kind   script.TypeKind
	offset uint32
}

type structLayout struct {
	name   string
	fields []structField
	size   uint32
}

func (s *structLayout) findField(name string) (structField, bool) {
	for _, f := range s.fields {
		if f.name == name {
			return f, true
		}
	}
	return structField{}, false
}
