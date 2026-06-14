package compiler_test

import (
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

	wat, err := compiler.CompileProgram(ast.(*script.Program))
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
