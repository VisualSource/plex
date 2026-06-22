package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/VisualSource/plex/internal/script"
	"github.com/VisualSource/plex/internal/script/compiler"
	binary_wasm "github.com/VisualSource/plex/internal/script/compiler/binary"
)

var file string
var output string

func main() {
	flag.StringVar(&file, "file", "", "file to parse")
	flag.StringVar(&output, "target", "wasm-wat", "output format: wasm-wat,wasm-bin")

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
		result, err := compiler.CompileProgram(ast)
		if err != nil {
			fmt.Printf("error: %s", err.Error())
			return
		}

		out := filepath.Join("./dist/", name+".wat")

		f, err := os.Create(out)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		defer f.Close()
		if _, err := f.WriteString(result); err != nil {
			fmt.Println(err.Error())
		}

		f.Sync()
	case "wasm-bin":
		result, err := binary_wasm.CompileProgram(ast)
		if err != nil {
			fmt.Printf("error: %s", err.Error())
			return
		}

		out := filepath.Join("./dist/", name+".wasm")
		if err := os.WriteFile(out, result, 0644); err != nil {
			fmt.Println(err.Error())
		}
	default:
		fmt.Printf("unknown output target: %s", output)
	}
}
