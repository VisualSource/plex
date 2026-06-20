package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/VisualSource/plex/internal/script"
	scriptinterpreter "github.com/VisualSource/plex/internal/script/script_interpreter"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
)

var target string
var source string

func readString(offset uint32, mem api.Memory) (string, error) {
	buf, _ := mem.Read(offset, 256)

	n := bytes.IndexByte(buf, 0)
	if n < 0 {
		return "", errors.New("failed to find null byte")
	}

	value := string(buf[:n])

	return value, nil
}

func main() {
	flag.StringVar(&target, "target", "interpreter", "run given file using interpreter or compiler")
	flag.StringVar(&source, "file", "", "path file file to run")

	flag.Parse()

	if source != "" {
		file, err := os.OpenFile(source, os.O_RDONLY, 0667)
		if err != nil {
			fmt.Printf("error: %s\n", err.Error())
			return
		}
		defer file.Close()

		switch target {
		case "wasm":
			ctx := context.Background()
			r := wazero.NewRuntime(ctx)
			defer r.Close(ctx)

			builder := r.NewHostModuleBuilder("env").NewFunctionBuilder()
			_, err := builder.WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
				mem := mod.Memory()
				offset := api.DecodeU32(stack[0])

				value, err := readString(offset, mem)
				if err != nil {
					fmt.Println(err)
					return
				}

				fmt.Println(value)
			}),
				[]api.ValueType{api.ValueTypeI32},
				[]api.ValueType{},
			).Export("print").Instantiate(ctx)
			if err != nil {
				panic(err)
			}

			wasm, err := io.ReadAll(file)
			if err != nil {
				fmt.Printf("error: %s", err.Error())
				return
			}

			module, err := r.Instantiate(ctx, wasm)
			if err != nil {
				fmt.Printf("error: %s", err.Error())
				return
			}
			defer module.Close(ctx)

			rr, err := module.ExportedFunction("count").Call(ctx, 10)
			if err != nil {
				fmt.Printf("error: %s", err.Error())
				return
			}

			fmt.Printf("%v\n", rr)

			/*fmt.Printf("string offset: %d\n", result[0])
			offset := api.DecodeU32(result[0])

			memory := module.ExportedMemory("memory")

			value, err := readString(offset, memory)
			if err != nil {
				panic(err)
			}

			fmt.Printf("%s\n", value)*/

		case "interpreter":
			ast, err := script.Parse(file)
			if err != nil {
				fmt.Printf("error: %s", err.Error())
				return
			}

			env := scriptinterpreter.NewGlobalEnv()

			result, err := scriptinterpreter.Eval(ast, env)
			if err != nil {
				fmt.Printf("%serror: %s%s\n", Red, err.Error(), Reset)
				return
			}

			fmt.Printf("%s%s%s\n", Cyan, result.String(), Reset)

		default:
			fmt.Println("invalid target")
			return
		}

		return
	}

	env := scriptinterpreter.NewGlobalEnv()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Printf("%sError reading input: %s %s\n", Red, err, Reset)
				break
			}

			continue
		}

		tokens, err := script.NewTokenizer(strings.NewReader(scanner.Text())).Tokenize()
		if err != nil {
			fmt.Printf("%serror: %s%s\n", Red, err.Error(), Reset)
			continue
		}

		parser := script.NewParser(tokens)
		ast, err := parser.Parse()
		if err != nil {
			fmt.Printf("%serror: %s%s\n", Red, err.Error(), Reset)
			continue
		}

		result, err := scriptinterpreter.Eval(ast, env)
		if err != nil {
			fmt.Printf("%serror: %s%s\n", Red, err.Error(), Reset)
			continue
		}

		fmt.Printf("%s%s%s\n", Cyan, result.String(), Reset)
	}

}
