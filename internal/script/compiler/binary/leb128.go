package binary_wasm

func AppendULEB128(buf []byte, v uint32) []byte {
	for {
		b := byte(v & 0x7F)
		v >>= 7
		if v != 0 {
			buf = append(buf, b|0x80)
		} else {
			return append(buf, b)
		}
	}
}

func AppendSLEB128(buf []byte, v int64) []byte {
	for {
		b := byte(v) & 0x7F
		signBit := b & 0x40
		v >>= 7
		done := (v == 0 && signBit == 0) || (v == -1 && signBit != 0)
		if done {
			return append(buf, b)
		}
		buf = append(buf, b|0x80)
	}
}
