package binary_wasm

import (
	"github.com/VisualSource/plex/internal/script"
	"github.com/VisualSource/plex/internal/script/typeschecker"
)

var magic = []byte{0x00, 0x61, 0x73, 0x6D}   // "\0asm"
var version = []byte{0x01, 0x00, 0x00, 0x00} // 1 (little-endian u32)

func CompileProgram(node *script.Program) ([]byte, error) {
	if err := typeschecker.Check(node); err != nil {
		return nil, err
	}

	strings := collectStrings(node)
	sigs, funcs, exports, err := collectSignatures(node)
	if err != nil {
		return nil, err
	}

	needsHeap := len(strings.entries) > 0 || programHasArrays(node)

	if needsHeap {
		exports = append(exports, exportEntry{name: "memory", kind: ExportMemory, idx: 0})
		exports = append(exports, exportEntry{
			name: "__heap_ptr", kind: ExportGlobal, idx: 0,
		})
	}

	funcIndexes := make(map[string]uint32, len(sigs))
	for i, s := range sigs {
		funcIndexes[s.name] = uint32(i)
	}

	var out []byte
	out = append(out, magic...)
	out = append(out, version...)

	if len(sigs) > 0 {
		out = append(out, encodeTypeSection(sigs)...)
		out = append(out, encodeFunctionSection(sigs)...)

		if needsHeap {
			out = append(out, encodeMemorySection()...)
			out = append(out, encodeGlobalSection(strings.next)...)
		}

		if len(exports) > 0 {
			out = append(out, encodeExportSection(exports)...)
		}

		code, err := encodeCodeSection(funcs, sigs, funcIndexes, strings)
		if err != nil {
			return nil, err
		}

		out = append(out, code...)

		if len(strings.entries) > 0 {
			out = append(out, encodeDataSection(strings)...)
		}
	}

	// sections will be appended here in later lessons
	return out, nil
}
