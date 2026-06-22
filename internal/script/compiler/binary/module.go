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

	sigs, funcs, err := collectSignatures(node)
	if err != nil {
		return nil, err
	}

	var out []byte
	out = append(out, magic...)
	out = append(out, version...)

	if len(sigs) > 0 {
		out = append(out, encodeTypeSection(sigs)...)
		out = append(out, encodeFunctionSection(sigs)...)

		code, err := encodeCodeSection(funcs)
		if err != nil {
			return nil, err
		}

		out = append(out, code...)
	}

	// sections will be appended here in later lessons
	return out, nil
}
