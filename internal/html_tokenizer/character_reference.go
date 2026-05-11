package html_tokenizer

import (
	"bytes"
	_ "embed"
	"encoding/gob"

	"github.com/acomagu/trie/v2"
)

type characterReferences struct {
	K [][]rune
	V [][]int
}

//go:generate go run ../../scripts/character_ref_to_bin.go ../../resources/character_reference.json character_reference.bin
//go:embed character_reference.bin
var characterReferencesRaw []byte
var characterReferenceTrie = createCharacterReferenceTrie()

func createCharacterReferenceTrie() trie.Tree[rune, []int] {
	dec := gob.NewDecoder(bytes.NewReader(characterReferencesRaw))

	var data characterReferences
	if err := dec.Decode(&data); err != nil {
		panic(err)
	}

	return trie.New(data.K, data.V)
}

// use a trie and longest match to find an match
// else returns nil if no value was found
func getCharacterReference(buffer *[]rune) ([]int, int) {
	trie := characterReferenceTrie

	var matchedCodepoint []int = nil
	var size int = -1 // number of chars used to find ref
	for idx, char := range *buffer {
		if trie = trie.TraceOne(char); trie == nil {
			break
		}

		if vv, ok := trie.Terminal(); ok {
			matchedCodepoint = vv
			size = idx + 1
		}
	}

	return matchedCodepoint, size
}
