package binary_wasm_test

import (
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/script"
	binary_wasm "github.com/VisualSource/plex/internal/script/compiler/binary"
	"github.com/tetratelabs/wazero"
)

func TestEndToEnd_FortyTwo(t *testing.T) {
	src := `
        fn forty_two(): i64 {
            return 42;
        }
    `

	ast, err := script.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}

	wasm, err := binary_wasm.CompileProgram(ast)
	if err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	mod, err := r.Instantiate(ctx, wasm)
	if err != nil {
		t.Fatalf("instantiate: %v", err)
	}

	got, err := mod.ExportedFunction("forty_two").Call(ctx)
	if err != nil {
		t.Fatalf("call: %v", err)
	}

	if got[0] != 42 {
		t.Fatalf("forty_two() = %d, want 42", got[0])
	}
}

func TestEndToEnd_Add(t *testing.T) {
	src := `
        fn add(a: i64, b: i64): i64 {
            return a + b;
        }
    `
	runOne(t, src, "add", []uint64{7, 35}, 42)
}

func TestEndToEnd_Sub(t *testing.T) {
	src := `
        fn sub3(a: i64, b: i64, c: i64): i64 {
            return a - b - c;
        }
    `
	// (100 - 50) - 8 = 42; if left/right were swapped you'd get
	// 100 - (50 - 8) = 58 — useful canary for the walk order.
	runOne(t, src, "sub3", []uint64{100, 50, 8}, 42)
}

func TestEndToEnd_Locals(t *testing.T) {
	src := `
        fn accumulate(a: i64, b: i64): i64 {
            let x: i64 = a + b;
            let y: i64 = x + 1;
            return y;
        }
    `
	runOne(t, src, "accumulate", []uint64{20, 21}, 42)
}

func TestEndToEnd_IfElse(t *testing.T) {
	src := `
        fn max(a: i64, b: i64): i64 {
            if a > b {
                return a;
            } else {
                return b;
            }
        }
    `
	runOne(t, src, "max", []uint64{42, 17}, 42) // takes then branch
	runOne(t, src, "max", []uint64{17, 42}, 42) // takes else branch
}

func TestEndToEnd_WhileSum(t *testing.T) {
	src := `
        fn sum(n: i64): i64 {
            let i: i64 = 1;
            let acc: i64 = 0;
            while i <= n {
                acc = acc + i;
                i = i + 1;
            }
            return acc;
        }
    `
	runOne(t, src, "sum", []uint64{10}, 55) // 1+2+…+10 = 55
	runOne(t, src, "sum", []uint64{0}, 0)   // zero iterations
	runOne(t, src, "sum", []uint64{1}, 1)   // exactly one iteration
}

func runOne(t *testing.T, src, fn string, args []uint64, want uint64) {
	t.Helper()
	ast, err := script.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	wasm, err := binary_wasm.CompileProgram(ast)
	if err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)
	mod, err := r.Instantiate(ctx, wasm)
	if err != nil {
		t.Fatalf("instantiate: %v", err)
	}

	got, err := mod.ExportedFunction(fn).Call(ctx, args...)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if got[0] != want {
		t.Fatalf("%s%v = %d, want %d", fn, args, got[0], want)
	}
}
