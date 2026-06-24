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

	imports := collectImports(node)
	strings := collectStrings(node)
	structs := collectStructs(node)

	sigs, funcs, exports, err := collectSignatures(node)
	if err != nil {
		return nil, err
	}

	needsHeap := len(strings.entries) > 0 || programHasHeapAllocation(node, structs)
	if needsHeap {
		exports = append(exports, exportEntry{name: "memory", kind: ExportMemory, idx: 0})
		exports = append(exports, exportEntry{
			name: "__heap_ptr", kind: ExportGlobal, idx: 0,
		})
	}

	importOffset := uint32(len(imports))
	funcIndexes := make(map[string]uint32, len(imports)+len(sigs))
	for i, imp := range imports {
		funcIndexes[imp.funcName] = uint32(i)
	}
	for i, s := range sigs {
		funcIndexes[s.name] = importOffset + uint32(i)
	}
	for i := range exports {
		if exports[i].kind == ExportFunc {
			exports[i].idx = funcIndexes[exports[i].name]
		}
	}

	// Build the type section: import types first, then local types.
	allSigs := make([]funcSig, 0, len(imports)+len(sigs))
	for _, imp := range imports {
		allSigs = append(allSigs, funcSig{
			name:    imp.funcName,
			params:  imp.params,
			results: imp.results,
		})
	}
	allSigs = append(allSigs, sigs...)

	var out []byte
	out = append(out, magic...)
	out = append(out, version...)

	if len(allSigs) > 0 {
		out = append(out, encodeTypeSection(allSigs)...)
		if len(imports) > 0 {
			out = append(out, encodeImportSection(imports)...) // NEW — before function section
		}

		out = append(out, encodeFunctionSection(sigs, importOffset)...)

		if needsHeap {
			out = append(out, encodeMemorySection()...)
			out = append(out, encodeGlobalSection(strings.next)...)
		}

		if len(exports) > 0 {
			out = append(out, encodeExportSection(exports)...)
		}

		code, err := encodeCodeSection(funcs, sigs, funcIndexes, strings, structs)
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
