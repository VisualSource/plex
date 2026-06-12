package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/VisualSource/plex/internal/script"
	scriptinterpreter "github.com/VisualSource/plex/internal/script/script_interpreter"
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

func main() {

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
