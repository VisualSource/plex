package scriptinterpreter_test

import (
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/script"
	scriptinterpreter "github.com/VisualSource/plex/internal/script/script_interpreter"
)

func TestEval(t *testing.T) {
	tokens, err := script.NewTokenizer(strings.NewReader(`
		fn mul(a: int, b: int): int {
			let result = 0;
			let i = 0;
			while i < b {
				result = result + a;
				i = i + 1;
			}

			return result;
		}
		mul(3,4);
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

	if v != "12" {
		t.Fatalf("was expecting to get 7 but get %s", v)
	}
}

func TestEvalGlobalEnv(t *testing.T) {
	tokens, err := script.NewTokenizer(strings.NewReader(`
		struct Point { x: int; y: int; }
		impl Point {
			fn sum(): int {
				return self.x + self.y;
			}
		}
		let p = Point(1,2);
		print(str(p.sum()));
	`)).Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	p := script.NewParser(tokens)
	ast, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}

	env := scriptinterpreter.NewGlobalEnv()
	_, err = scriptinterpreter.Eval(ast, env)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEvalWhileBreak(t *testing.T) {
	tokens, err := script.NewTokenizer(strings.NewReader(`
		let mut i = 0;
		while 0 == 0 {
			break;
			i++;
		} 
		i;
	`)).Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	p := script.NewParser(tokens)
	ast, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}

	env := scriptinterpreter.NewGlobalEnv()
	body, err := scriptinterpreter.Eval(ast, env)
	if err != nil {
		t.Fatal(err)
	}

	if body.String() != "0" {
		t.Fatal("was expecting value to be zero")
	}
}
