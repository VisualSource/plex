package binary_wasm_test

import (
	"bytes"
	"testing"

	binary_wasm "github.com/VisualSource/plex/internal/script/compiler/binary"
)

func TestULEB128(t *testing.T) {
	cases := []struct {
		in   uint32
		want []byte
	}{
		{0, []byte{0x00}},
		{63, []byte{0x3F}},
		{127, []byte{0x7F}},
		{128, []byte{0x80, 0x01}},
		{624485, []byte{0xE5, 0x8E, 0x26}},
	}
	for _, c := range cases {
		got := binary_wasm.AppendULEB128(nil, c.in)
		if !bytes.Equal(got, c.want) {
			t.Errorf("uleb(%d) = % x, want % x", c.in, got, c.want)
		}
	}
}
