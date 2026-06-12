package scriptinterpreter_test

import (
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/script"
	scriptinterpreter "github.com/VisualSource/plex/internal/script/script_interpreter"
)

func TestEval(t *testing.T) {
	tokens, err := script.NewTokenizer(strings.NewReader(`
		fn add(a: int, b: int): int {
			return a + b;
		}
		add(3,4);
	`)).Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	p := script.NewParser(tokens)
	ast, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}

	env := scriptinterpreter.NewEnvironment(nil)
	output, err := scriptinterpreter.Eval(ast, env)
	if err != nil {
		t.Fatal(err)
	}

	v := output.String()

	if v != "7" {
		t.Fatalf("was expecting to get 7 but get %s", v)
	}
}
