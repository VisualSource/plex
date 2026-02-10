package html_tokenizer

import (
	"bytes"
	_ "embed"
	"encoding/gob"

	"github.com/acomagu/trie/v2"
)

type characterReferences struct {
	Keys   [][]rune
	Values []int
}

//go:embed character_reference.bin
var characterReferencesRaw []byte
var characterReferenceTrie = createCharacterReferenceTrie()

func createCharacterReferenceTrie() trie.Tree[rune, int] {
	buffer := bytes.NewReader(characterReferencesRaw)
	dec := gob.NewDecoder(buffer)

	var data characterReferences
	err := dec.Decode(&data)
	if err != nil {
		panic(err)
	}

	return trie.New(data.Keys, data.Values)
}

// use a trie and longest match to find an match
// else return a -1 if no value was found
func getCharacterReference(buffer *[]rune) (int, int) {
	trie := characterReferenceTrie

	var matchedCodepoint int = -1
	var size int = -1
	for idx, char := range *buffer {
		if trie = trie.TraceOne(char); trie == nil {
			break
		}

		if vv, ok := trie.Terminal(); ok {
			matchedCodepoint = vv
			size = idx
		}
	}

	return matchedCodepoint, size
}
