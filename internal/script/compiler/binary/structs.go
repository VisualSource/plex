package binary_wasm

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
