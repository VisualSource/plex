package compiler_test

import (
	"os"
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/script"
	"github.com/VisualSource/plex/internal/script/compiler"
)

func TestCompilerFunctionCall(t *testing.T) {
	src := `
		fn add(a: float, b: float): int { return a + b; }
		fn double(x: float): float { return add(x,x); }	
	`

	ast, err := script.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}

	wat, err := compiler.CompileProgram(ast)
	if err != nil {
		t.Fatal(err)
	}

	checks := []string{
		"(func $add",
		"(func $double",
		"call $add",
		"(export \"add\"",
		"(export \"double\"",
		"f64.add",
	}

	for _, want := range checks {
		if !strings.Contains(wat, want) {
			t.Errorf("WAT missing %q\n\nGot:\n%s", want, wat)
		}
	}
}

func TestStructGenAndCtor(t *testing.T) {
	src := `
		struct Point { x: float; y: float; }

		fn make_point(x: float, y: float): Point {
			return Point(x, y);
		}

		fn get_x(p: Point): float {
			return p.x;
		}
	`

	ast, err := script.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}

	result, err := compiler.CompileProgram(ast)
	if err != nil {
		t.Fatal(err)
	}

	snapshot, err := os.ReadFile("testdata/struct.wat")
	if err != nil {
		t.Fatal(err)
	}

	if result != string(snapshot) {
		t.Fatal("result != snapshot")
	}
}
