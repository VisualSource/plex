package binary_wasm

type funcSig struct {
	name    string
	params  []byte
	results []byte
}

type exportEntry struct {
	name string
	kind byte
	idx  uint32
}
