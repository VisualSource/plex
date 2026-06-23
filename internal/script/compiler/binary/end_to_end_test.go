package binary_wasm_test

import (
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/script"
	binary_wasm "github.com/VisualSource/plex/internal/script/compiler/binary"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
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

func TestEndToEnd_FunctionCall(t *testing.T) {
	src := `
        fn square(x: i64): i64 {
            return x * x;
        }
        fn sum_squares(a: i64, b: i64): i64 {
            return square(a) + square(b);
        }
    `
	runOne(t, src, "sum_squares", []uint64{3, 4}, 25)   // 3²+4²=25
	runOne(t, src, "sum_squares", []uint64{5, 12}, 169) // 5²+12²=169=13²
	runOne(t, src, "square", []uint64{7}, 49)           // direct call too
}

func TestEndToEnd_Recursion(t *testing.T) {
	src := `
        fn fib(n: i64): i64 {
            if n < 2 { return n; }
            return fib(n - 1) + fib(n - 2);
        }
    `
	runOne(t, src, "fib", []uint64{0}, 0)
	runOne(t, src, "fib", []uint64{1}, 1)
	runOne(t, src, "fib", []uint64{10}, 55)
}

func TestEndToEnd_CallAsStatement(t *testing.T) {
	src := `
        fn square(x: i64): i64 {
            return x * x;
        }
        fn warmup(n: i64): i64 {
            square(n);          // result discarded — pure waste, but legal
            return n + 1;
        }
    `
	runOne(t, src, "warmup", []uint64{10}, 11)
}

func TestEndToEnd_ShortCircuitOr(t *testing.T) {
	src := `
        fn safe_or(a: bool, b: i64): bool {
            return a || (10 / b) > 0;
        }
    `
	// a=true: RHS must not be evaluated, even though 10/0 would trap
	runOne(t, src, "safe_or", []uint64{1, 0}, 1)

	// a=false: RHS runs. 10/5 = 2, 2 > 0 = true
	runOne(t, src, "safe_or", []uint64{0, 5}, 1)

	// a=false: RHS runs. 10/100 = 0, 0 > 0 = false
	runOne(t, src, "safe_or", []uint64{0, 100}, 0)
}

func TestEndToEnd_UnaryMinus(t *testing.T) {
	src := `
        fn neg_then_add(x: i64, y: i64): i64 {
            return -x + y;
        }
        fn double_neg(x: i64): i64 {
            return -(-x);          // should equal x
        }
        fn unary_plus(x: i64): i64 {
            return +x;             // no-op
        }
    `
	runOne(t, src, "neg_then_add", []uint64{10, 3}, api.EncodeI64(-7))
	runOne(t, src, "double_neg", []uint64{42}, 42)
	runOne(t, src, "unary_plus", []uint64{7}, 7)
}

func TestEndToEnd_StringLiteral(t *testing.T) {
	src := `
        fn greet(): string {
            return "hello";
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

	got, err := mod.ExportedFunction("greet").Call(ctx)
	if err != nil {
		t.Fatalf("call: %v", err)
	}

	ptr := api.DecodeU32(got[0])
	mem := mod.Memory()

	n, _ := mem.ReadUint32Le(ptr)
	if n != 5 {
		t.Fatalf("length = %d, want 5", n)
	}

	data, _ := mem.Read(ptr+4, n)
	if string(data) != "hello" {
		t.Fatalf("data = %q, want %q", string(data), "hello")
	}
}

func TestEndToEnd_StringLen(t *testing.T) {
	src := `
        fn measure(): int {
            return "hello".len();
        }
        fn measure_world(): int {
            return "wonderful".len();
        }
        fn measure_empty(): int {
            return "".len();
        }
    `
	runOne(t, src, "measure", nil, 5)
	runOne(t, src, "measure_world", nil, 9)
	runOne(t, src, "measure_empty", nil, 0)
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
