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
	args := os.Args[1:]

	inputPath := args[0]
	outputPathRoot := args[1]

	gp, err := filepath.Abs("./")
	if err != nil {
		panic(err)
	}

	source, err := os.ReadFile(inputPath)
	if err != nil {
		panic(err)
	}

	references := make(map[string]characterReference, 2231)

	if err = json.Unmarshal(source, &references); err != nil {
		panic(err)
	}

	var output struct {
		K [][]rune
		V [][]int
	}
	for key, value := range references {
		v, _ := strings.CutPrefix(key, "&")
		output.K = append(output.K, []rune(v))
		output.V = append(output.V, value.Codepoints)
	}

	outputPath := filepath.Join(gp, outputPathRoot)
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
