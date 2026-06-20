package compiler_test

import (
	"context"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/script"
	"github.com/VisualSource/plex/internal/script/compiler"
	"github.com/kr/pretty"
	"github.com/tetratelabs/wazero"
)

func validateFromSnapshot(t *testing.T, src string, file string) {
	t.Helper()
	ast, err := script.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}

	result, err := compiler.CompileProgram(ast)
	if err != nil {
		t.Fatal(err)
	}

	snapshot, err := os.ReadFile("testdata/" + file)
	if err != nil {
		t.Fatal(err)
	}

	if result != string(snapshot) {
		pretty.Ldiff(t, result, string(snapshot))
		t.Fail()
	}
}

type testRun struct {
	fn   string
	args []uint64
	want []uint64
}

func validateByRun(t *testing.T, source string, tests []testRun, configEnv func(ctx context.Context, r wazero.Runtime)) {
	t.Helper()

	ctx := t.Context()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	configEnv(ctx, r)

	ast, err := script.Parse(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}

	result, err := compiler.CompileProgram(ast)
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()

	tempName := t.Name() + "*.wat"
	file, err := os.CreateTemp(tempDir, tempName)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := file.WriteString(result); err != nil {
		t.Fatal(err)
	}

	inputFile := file.Name()
	outputFile := strings.Replace(file.Name(), "wat", "wasm", 1)

	cmd := exec.CommandContext(ctx, "wat2wasm", inputFile, "-o", outputFile) // when able replace this with binrary wasm so we do have to pass it to wat2wasm
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s\n %s", string(cmdOutput), err)
	}

	wasm, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	module, err := r.Instantiate(ctx, wasm)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range tests {
		result, err := module.ExportedFunction(test.fn).Call(ctx, test.args...)
		if err != nil {
			t.Fatal(err)
		}

		if slices.Compare(test.want, result) != 0 {
			pretty.Ldiff(t, test.want, result)
			t.Fail()
		}
	}
}
