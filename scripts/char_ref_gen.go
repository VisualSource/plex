package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type characterReference struct {
	Codepoints []int  `json:"codepoints"`
	Characters string `json:"characters"`
}

type CharacterReferences struct {
	Keys   [][]rune
	Values []int
}

func main() {
	gp, err := filepath.Abs("./")
	if err != nil {
		panic(err)
	}

	fmt.Printf("GOPATH: %s\n", gp)

	cr := filepath.Join(gp, "resources/character_reference.json")

	source, err := os.ReadFile(cr)
	if err != nil {
		panic(err)
	}

	references := make(map[string]characterReference, 2231)

	err = json.Unmarshal(source, &references)
	if err != nil {
		panic(err)
	}

	var output CharacterReferences

	for key, value := range references {

		v, _ := strings.CutPrefix(key, "&")

		output.Keys = append(output.Keys, []rune(v))
		output.Values = append(output.Values, value.Codepoints[0])
	}

	outputPath := filepath.Join(gp, "internal/html_tokenizer/character_reference.bin")
	fmt.Printf("Output file: %s\n", outputPath)

	file, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}

	enc := gob.NewEncoder(file)
	err = enc.Encode(output)
	if err != nil {
		panic(err)
	}
}
