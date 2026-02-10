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

	if err = json.Unmarshal(source, &references); err != nil {
		panic(err)
	}

	var output struct {
		K [][]rune
		V []int
	}
	for key, value := range references {
		v, _ := strings.CutPrefix(key, "&")
		output.K = append(output.K, []rune(v))
		output.V = append(output.V, value.Codepoints[0])
	}

	outputPath := filepath.Join(gp, "internal/html_tokenizer/character_reference.bin")
	fmt.Printf("Output file: %s\n", outputPath)

	file, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}

	enc := gob.NewEncoder(file)
	if err = enc.Encode(output); err != nil {
		panic(err)
	}
}
