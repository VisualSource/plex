package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/VisualSource/plex/internal/script"
	"github.com/VisualSource/plex/internal/script/compiler"
)

var file string
var output string

func main() {
	flag.StringVar(&file, "file", "", "file to parse")
	flag.StringVar(&output, "target", "wasm-wat", "ouput format: wasm-wat,wasm-bin")

	flag.Parse()

	filename := filepath.Base(file)

	p := strings.Split(filename, ".")
	name := p[0]

	file, err := os.OpenFile(file, os.O_RDONLY, 0666)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		return
	}

	ast, err := script.Parse(file)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		return
	}

	switch output {
	case "wasm-wat":
		result, err := compiler.CompileProgram(ast.(*script.Program))
		if err != nil {
			fmt.Printf("error: %s", err.Error())
			return
		}

		if err := os.WriteFile(name+".wat", []byte(result), 0666); err != nil {
			fmt.Printf("error: %s", err.Error())
			return
		}

	default:
		fmt.Printf("unknown output target: %s", output)
	}
}
