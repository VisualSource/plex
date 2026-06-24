package binary_wasm_test

import (
	"context"
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/script"
	binary_wasm "github.com/VisualSource/plex/internal/script/compiler/binary"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func TestEndToEnd_FortyTwo(t *testing.T) {
	src := `
        export fn forty_two(): i64 {
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
        export fn add(a: i64, b: i64): i64 {
            return a + b;
        }
    `
	runOne(t, src, "add", []uint64{7, 35}, 42)
}

func TestEndToEnd_Sub(t *testing.T) {
	src := `
        export fn sub3(a: i64, b: i64, c: i64): i64 {
            return a - b - c;
        }
    `
	// (100 - 50) - 8 = 42; if left/right were swapped you'd get
	// 100 - (50 - 8) = 58 — useful canary for the walk order.
	runOne(t, src, "sub3", []uint64{100, 50, 8}, 42)
}

func TestEndToEnd_Locals(t *testing.T) {
	src := `
        export fn accumulate(a: i64, b: i64): i64 {
            let x: i64 = a + b;
            let y: i64 = x + 1;
            return y;
        }
    `
	runOne(t, src, "accumulate", []uint64{20, 21}, 42)
}

func TestEndToEnd_IfElse(t *testing.T) {
	src := `
        export fn max(a: i64, b: i64): i64 {
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
        export fn sum(n: i64): i64 {
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
        export fn square(x: i64): i64 {
            return x * x;
        }
        export fn sum_squares(a: i64, b: i64): i64 {
            return square(a) + square(b);
        }
    `
	runOne(t, src, "sum_squares", []uint64{3, 4}, 25)   // 3²+4²=25
	runOne(t, src, "sum_squares", []uint64{5, 12}, 169) // 5²+12²=169=13²
	runOne(t, src, "square", []uint64{7}, 49)           // direct call too
}

func TestEndToEnd_Recursion(t *testing.T) {
	src := `
        export fn fib(n: i64): i64 {
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
        export fn square(x: i64): i64 {
            return x * x;
        }
        export fn warmup(n: i64): i64 {
            square(n);          // result discarded — pure waste, but legal
            return n + 1;
        }
    `
	runOne(t, src, "warmup", []uint64{10}, 11)
}

func TestEndToEnd_ShortCircuitOr(t *testing.T) {
	src := `
        export fn safe_or(a: bool, b: i64): bool {
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
        export fn neg_then_add(x: i64, y: i64): i64 {
            return -x + y;
        }
        export fn double_neg(x: i64): i64 {
            return -(-x);          // should equal x
        }
        export fn unary_plus(x: i64): i64 {
            return +x;             // no-op
        }
    `
	runOne(t, src, "neg_then_add", []uint64{10, 3}, api.EncodeI64(-7))
	runOne(t, src, "double_neg", []uint64{42}, 42)
	runOne(t, src, "unary_plus", []uint64{7}, 7)
}

func TestEndToEnd_StringLiteral(t *testing.T) {
	src := `
        export fn greet(): string {
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
        export fn measure(): int {
            return "hello".len();
        }
        export fn measure_world(): int {
            return "wonderful".len();
        }
        export fn measure_empty(): int {
            return "".len();
        }
    `
	runOne(t, src, "measure", nil, 5)
	runOne(t, src, "measure_world", nil, 9)
	runOne(t, src, "measure_empty", nil, 0)
}

func TestEndToEnd_I32Comparisons(t *testing.T) {
	// Plex's typechecker doesn't coerce integer literals to i32, so we exercise
	// the new opcodes through pure parameter-driven comparisons.
	src := `
        export fn lt_i32(a: i32, b: i32): bool { return a < b; }
        export fn gt_i32(a: i32, b: i32): bool { return a > b; }
        export fn le_i32(a: i32, b: i32): bool { return a <= b; }
        export fn ge_i32(a: i32, b: i32): bool { return a >= b; }
    `
	runOne(t, src, "lt_i32", []uint64{3, 7}, 1) // 3 <  7
	runOne(t, src, "lt_i32", []uint64{7, 3}, 0)
	runOne(t, src, "lt_i32", []uint64{3, 3}, 0) // strict — catches lt vs le confusion
	runOne(t, src, "gt_i32", []uint64{7, 3}, 1) // 7 >  3
	runOne(t, src, "gt_i32", []uint64{3, 7}, 0)
	runOne(t, src, "le_i32", []uint64{3, 3}, 1) // 3 <= 3
	runOne(t, src, "le_i32", []uint64{4, 3}, 0)
	runOne(t, src, "ge_i32", []uint64{3, 3}, 1) // 3 >= 3
	runOne(t, src, "ge_i32", []uint64{2, 3}, 0)
}

func TestEndToEnd_F64Comparisons(t *testing.T) {
	src := `
        export fn classify(x: f64): i64 {
            if x < 0.0 { return -1; }
            if x > 0.0 { return  1; }
            return 0;
        }
        export fn near_one(x: f64): bool {
            return x >= 0.99 && x <= 1.01;
        }
    `
	runOne(t, src, "classify", []uint64{api.EncodeF64(-3.5)}, api.EncodeI64(-1))
	runOne(t, src, "classify", []uint64{api.EncodeF64(0.0)}, 0)
	runOne(t, src, "classify", []uint64{api.EncodeF64(42.0)}, 1)

	runOne(t, src, "near_one", []uint64{api.EncodeF64(1.00)}, 1)
	runOne(t, src, "near_one", []uint64{api.EncodeF64(0.95)}, 0)
}

func TestEndToEnd_HeapPointer(t *testing.T) {
	src := `
        export fn label(): string { return "hello"; }
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

	g := mod.ExportedGlobal("__heap_ptr")
	if g == nil {
		t.Fatalf("__heap_ptr not exported")
	}

	got := api.DecodeU32(g.Get())
	// "hello" = 4 length bytes + 5 ASCII bytes = 9 bytes at offset 0.
	// Heap starts immediately after.
	if got != 9 {
		t.Fatalf("__heap_ptr = %d, want 9", got)
	}
}

func TestEndToEnd_HeapPointerMultipleStrings(t *testing.T) {
	src := `
        export fn one(): string { return "hi"; }
        export fn two(): string { return "world"; }
    `
	// "hi"    → 4 + 2 = 6 bytes at offset 0..5
	// "world" → 4 + 5 = 9 bytes at offset 6..14
	// heap starts at 15
	ast, _ := script.Parse(strings.NewReader(src))
	wasm, _ := binary_wasm.CompileProgram(ast)

	ctx := t.Context()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)
	mod, err := r.Instantiate(ctx, wasm)
	if err != nil {
		t.Fatalf("instantiate: %v", err)
	}

	got := api.DecodeU32(mod.ExportedGlobal("__heap_ptr").Get())
	if got != 15 {
		t.Fatalf("__heap_ptr = %d, want 15", got)
	}
}

func TestEndToEnd_ArrayLiteral(t *testing.T) {
	src := `
        export fn make_arr(): int[] {
            return [10, 20, 30];
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

	got, err := mod.ExportedFunction("make_arr").Call(ctx)
	if err != nil {
		t.Fatalf("call: %v", err)
	}

	ptr := api.DecodeU32(got[0])
	mem := mod.Memory()

	n, _ := mem.ReadUint32Le(ptr)
	if n != 3 {
		t.Fatalf("length = %d, want 3", n)
	}

	want := []uint64{10, 20, 30}
	for i, w := range want {
		v, _ := mem.ReadUint64Le(ptr + 4 + uint32(i)*8)
		if v != w {
			t.Fatalf("arr[%d] = %d, want %d", i, v, w)
		}
	}
}

func TestEndToEnd_ArrayLen(t *testing.T) {
	src := `
        export fn three(): int {
            let arr = [10, 20, 30];
            return arr.len();
        }
        export fn empty(): int {
            let arr: int[] = [];
            return arr.len();
        }
    `
	runOne(t, src, "three", nil, 3)
	runOne(t, src, "empty", nil, 0)
}

func TestEndToEnd_ArrayIndex(t *testing.T) {
	src := `
        export fn first(): int {
            let arr = [10, 20, 30];
            return arr[0];
        }
        export fn middle(): int {
            let arr = [10, 20, 30];
            return arr[1];
        }
        export fn last(): int {
            let arr = [10, 20, 30];
            return arr[2];
        }
        export fn sum_all(): int {
            let arr = [10, 20, 30];
            return arr[0] + arr[1] + arr[2];
        }
    `
	runOne(t, src, "first", nil, 10)
	runOne(t, src, "middle", nil, 20)
	runOne(t, src, "last", nil, 30)
	runOne(t, src, "sum_all", nil, 60)
}

func TestEndToEnd_Sum(t *testing.T) {
	src := `
       export fn total(): int {
			let arr = [1, 2, 3, 4, 5];
			let i: int = 0;
			let sum: int = 0;
			while i < arr.len() {
				sum = sum + arr[i];
				i = i + 1;
			}
			return sum;
		}
    `
	runOne(t, src, "total", nil, 15)
}

func TestEndToEnd_ArrayAssignment(t *testing.T) {
	src := `
        export fn write_then_read(): int {
            let arr = [10, 20, 30];
            arr[1] = 99;
            return arr[1];
        }
        export fn rotate(): int {
            let arr = [1, 2, 3, 4, 5];
            arr[0] = arr[4];
            arr[4] = 1;
            return arr[0] + arr[4]; // 5 + 1 = 6
        }
        export fn sort_two(): int {
            let arr = [20, 10];
            if arr[0] > arr[1] {
                let tmp = arr[0];
                arr[0] = arr[1];
                arr[1] = tmp;
            }
            return arr[0] * 10 + arr[1]; // 10*10 + 20 = 120
        }
    `
	runOne(t, src, "write_then_read", nil, 99)
	runOne(t, src, "rotate", nil, 6)
	runOne(t, src, "sort_two", nil, 120)
}

func TestEndToEnd_BoundsInBounds(t *testing.T) {
	src := `
        export fn get_at(): int {
            let arr = [10, 20, 30];
            return arr[2];
        }
        export fn set_at(): int {
            let arr = [10, 20, 30];
            arr[0] = 99;
            return arr[0];
        }
    `
	runOne(t, src, "get_at", nil, 30)
	runOne(t, src, "set_at", nil, 99)
}

func TestEndToEnd_BoundsOutOfRange(t *testing.T) {
	src := `
        export fn read_oob(): int {
            let arr = [10, 20, 30];
            return arr[100];
        }
        export fn write_oob(): int {
            let arr = [10, 20, 30];
            arr[100] = 99;
            return 0;
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

	_, err = mod.ExportedFunction("read_oob").Call(ctx)
	if err == nil {
		t.Fatalf("expected a trap on out-of-bounds read; got nil error")
	}

	_, err = mod.ExportedFunction("write_oob").Call(ctx)
	if err == nil {
		t.Fatalf("expected a trap on out-of-bounds write; got nil error")
	}
}

func TestEndToEnd_StructFields(t *testing.T) {
	src := `
        struct Point {
            x: f64;
            y: f64;
        }
        export fn make_point_x(): f64 {
            let p = Point(3.0, 4.0);
            return p.x;
        }
        export fn make_point_y(): f64 {
            let p = Point(3.0, 4.0);
            return p.y;
        }
        export fn distance_squared(): f64 {
            let p = Point(3.0, 4.0);
            return p.x * p.x + p.y * p.y;  // 9 + 16 = 25
        }
        export fn mutate_y(): f64 {
            let p = Point(3.0, 4.0);
            p.y = 99.0;
            return p.y;
        }
    `
	runOne(t, src, "make_point_x", nil, api.EncodeF64(3.0))
	runOne(t, src, "make_point_y", nil, api.EncodeF64(4.0))
	runOne(t, src, "distance_squared", nil, api.EncodeF64(25.0))
	runOne(t, src, "mutate_y", nil, api.EncodeF64(99.0))
}

func TestEndToEnd_StructMethods(t *testing.T) {
	src := `
        struct Point {
            x: f64;
            y: f64;
        }
        impl Point {
            fn distance_squared(): f64 {
                return self.x * self.x + self.y * self.y;
            }
            fn scale(factor: f64): f64 {
                return self.x * factor + self.y * factor;
            }
        }
        export fn d_sq(): f64 {
            let p = Point(3.0, 4.0);
            return p.distance_squared();  // 9 + 16 = 25
        }
        export fn scaled(): f64 {
            let p = Point(2.0, 3.0);
            return p.scale(10.0);          // 20 + 30 = 50
        }
    `
	runOne(t, src, "d_sq", nil, api.EncodeF64(25.0))
	runOne(t, src, "scaled", nil, api.EncodeF64(50.0))
}

func TestEndToEnd_MutatingMethod(t *testing.T) {
	src := `
        struct Vec2 {
            x: f64;
            y: f64;
        }
        impl Vec2 {
            fn translate(dx: f64, dy: f64) {
                self.x = self.x + dx;
                self.y = self.y + dy;
            }
            fn sum(): f64 {
                return self.x + self.y;
            }
        }
        export fn run(): f64 {
            let v = Vec2(1.0, 2.0);
            v.translate(10.0, 20.0);
            return v.sum();
        }
    `
	// (1+10) + (2+20) = 33
	runOne(t, src, "run", nil, api.EncodeF64(33.0))
}

func TestEndToEnd_MethodCallsMethod(t *testing.T) {
	src := `
        struct Circle {
            r: f64;
        }
        impl Circle {
            fn area_approx(): f64 {
                return self.r * self.r * 3.0;
            }
            fn double_area(): f64 {
                return self.area_approx() * 2.0;
            }
        }
        export fn run(): f64 {
            let c = Circle(5.0);
            return c.double_area();
        }
    `
	// 5*5*3 * 2 = 150
	runOne(t, src, "run", nil, api.EncodeF64(150.0))
}

func TestEndToEnd_ArrayOfStructs(t *testing.T) {
	src := `
        struct Point { x: f64; y: f64; }
        export fn run(): f64 {
            let pts = [Point(1.0, 2.0), Point(10.0, 20.0)];
            let p = pts[1];
            return p.x + p.y;
        }
    `
	// pts[1] = Point(10, 20) → x+y = 30
	runOne(t, src, "run", nil, api.EncodeF64(30.0))
}

func TestEndToEnd_ArrayOfStructsMethods(t *testing.T) {
	src := `
        struct Vec2 { x: f64; y: f64; }
        impl Vec2 { fn magnitude_sq(): f64 { return self.x * self.x + self.y * self.y; } }
        export fn run(): f64 {
            let vs: Vec2[] = [Vec2(3.0, 4.0), Vec2(5.0, 12.0)];
            return vs[0].magnitude_sq() + vs[1].magnitude_sq();
        }
    `
	// 9+16=25, 25+144=169 → 25+169=194
	runOne(t, src, "run", nil, api.EncodeF64(194.0))
}

func TestEndToEnd_Ternary(t *testing.T) {
	src := `
        export fn pick(cond: bool, a: i64, b: i64): i64 {
            return cond ? a : b;
        }
        export fn ternary_arithmetic(x: i64): i64 {
            return (x > 0 ? x : -x) * 2;
        }
    `
	runOne(t, src, "pick", []uint64{1, 10, 20}, 10) // true  → a
	runOne(t, src, "pick", []uint64{0, 10, 20}, 20) // false → b
	runOne(t, src, "ternary_arithmetic", []uint64{api.EncodeI64(-5)}, 10)
	runOne(t, src, "ternary_arithmetic", []uint64{3}, 6)
}

func TestEndToEnd_TypeCast(t *testing.T) {
	src := `
        export fn i64_to_f64(x: i64): f64 { return x as f64; }
        export fn f64_to_i64(x: f64): i64 { return x as i64; }
        export fn i64_to_i32(x: i64): i32 { return x as i32; }
    `
	runOne(t, src, "i64_to_f64", []uint64{7}, api.EncodeF64(7.0))
	runOne(t, src, "f64_to_i64", []uint64{api.EncodeF64(3.9)}, 3) // truncates
	runOne(t, src, "f64_to_i64", []uint64{api.EncodeF64(-2.7)}, api.EncodeI64(-2))
	runOne(t, src, "i64_to_i32", []uint64{42}, 42)
}

func TestEndToEnd_ArrayAppend(t *testing.T) {
	src := `
        export fn grow(): int {
            let arr: int[] = [1, 2, 3];
            arr.append(4);
            return arr.len();
        }
        export fn grow_read(): int {
            let arr: int[] = [10, 20];
            arr.append(30);
            return arr[2];
        }
        export fn grow_twice(): int {
            let arr: int[] = [1];
            arr.append(2);
            arr.append(3);
            return arr[0] + arr[1] + arr[2]; // 1 + 2 + 3 = 6
        }
    `
	runOne(t, src, "grow", nil, 4)
	runOne(t, src, "grow_read", nil, 30)
	runOne(t, src, "grow_twice", nil, 6)
}

func TestEndToEnd_ArrayRemove(t *testing.T) {
	src := `
        export fn remove_middle(): int {
            let arr: int[] = [10, 20, 30];
            let r = arr.remove(1);
            return r;
        }
        export fn remove_shifts(): int {
            let arr: int[] = [10, 20, 30];
            arr.remove(0);
            return arr[0];
        }
        export fn remove_len(): int {
            let arr: int[] = [10, 20, 30];
            arr.remove(2);
            return arr.len();
        }
    `
	runOne(t, src, "remove_middle", nil, 20) // returns the element
	runOne(t, src, "remove_shifts", nil, 20) // arr[0] becomes 20 after remove(0)
	runOne(t, src, "remove_len", nil, 2)     // length decremented
}

func TestEndToEnd_Import_Print(t *testing.T) {
	src := `
        import print from "plex:console";
        export fn say_hello() {
            print("hello");
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

	var gotPtr uint32
	printCalled := false
	_, err = r.NewHostModuleBuilder("plex:console").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, ptr uint32) {
			printCalled = true
			gotPtr = ptr
		}).
		Export("print").
		Instantiate(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := r.Instantiate(ctx, wasm)
	if err != nil {
		t.Fatalf("instantiate: %v", err)
	}

	_, err = mod.ExportedFunction("say_hello").Call(ctx)
	if err != nil {
		t.Fatalf("call: %v", err)
	}

	if !printCalled {
		t.Fatal("print was never called")
	}

	// Read the string out of WASM memory to verify content
	mem := mod.Memory()
	n, _ := mem.ReadUint32Le(gotPtr)
	data, _ := mem.Read(gotPtr+4, n)
	if string(data) != "hello" {
		t.Fatalf("got %q, want \"hello\"", string(data))
	}
}

func TestExport_VisibilityBoundary(t *testing.T) {
	src := `
        export fn public_fn(): i64 { return 1; }
               fn private_fn(): i64 { return 2; }
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

	if mod.ExportedFunction("public_fn") == nil {
		t.Fatal("public_fn should be exported")
	}
	if mod.ExportedFunction("private_fn") != nil {
		t.Fatal("private_fn should NOT be exported")
	}
}

func TestEndToEnd_StringConcat(t *testing.T) {
	src := `
        export fn greet(): string {
            return "hello" + " world";
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
	data, _ := mem.Read(ptr+4, n)
	if string(data) != "hello world" {
		t.Fatalf("got %q, want \"hello world\"", string(data))
	}
}

func TestEndToEnd_StringConcatVar(t *testing.T) {
	src := `
        export fn build(): string {
            let a: string = "foo";
            let b: string = "bar";
            return a + b;
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

	got, err := mod.ExportedFunction("build").Call(ctx)
	if err != nil {
		t.Fatalf("call: %v", err)
	}

	ptr := api.DecodeU32(got[0])
	mem := mod.Memory()
	n, _ := mem.ReadUint32Le(ptr)
	data, _ := mem.Read(ptr+4, n)
	if string(data) != "foobar" {
		t.Fatalf("got %q, want \"foobar\"", string(data))
	}
}

func TestEndToEnd_StringEq(t *testing.T) {
	src := `
        export fn same(): bool   { return "hello" == "hello"; }
        export fn diff(): bool   { return "hello" == "world"; }
        export fn shorter(): bool { return "hi" == "hello"; }
    `
	runOne(t, src, "same", nil, 1)
	runOne(t, src, "diff", nil, 0)
	runOne(t, src, "shorter", nil, 0)
}

func TestEndToEnd_StringNe(t *testing.T) {
	src := `
        export fn same(): bool { return "hello" != "hello"; }
        export fn diff(): bool { return "hello" != "world"; }
    `
	runOne(t, src, "same", nil, 0)
	runOne(t, src, "diff", nil, 1)
}

func TestEndToEnd_Break(t *testing.T) {
	src := `
        export fn find_first(): int {
			let arr = [10, 20, 30, 40, 50];
			let i: int = 0;
			let found: int = -1;
			while i < arr.len() {
				if arr[i] > 25 {
					found = arr[i];
					break;
				}
				i = i + 1;
			}
			return found;
		}
    `
	runOne(t, src, "find_first", nil, 30)
}

func TestEndToEnd_Continue(t *testing.T) {
	src := `
        export fn sum_evens(n: i64): i64 {
			let i: i64 = 0;
			let sum: i64 = 0;
			while i < n {
				i = i + 1;
				if (i % 2) != 0 {
					continue;
				}
				sum = sum + i;
			}
			return sum;
		}
    `
	runOne(t, src, "sum_evens", []uint64{10}, 30)
}

func TestEndToEnd_Not(t *testing.T) {
	src := `
        export fn not_true(): bool  { return !true; }
        export fn not_false(): bool { return !false; }
        export fn double_not(x: bool): bool { return !!x; }
        export fn not_eq(a: i64, b: i64): bool { return !(a == b); }
    `
	runOne(t, src, "not_true", nil, 0)           // !true  = false
	runOne(t, src, "not_false", nil, 1)          // !false = true
	runOne(t, src, "double_not", []uint64{1}, 1) // !!true = true
	runOne(t, src, "double_not", []uint64{0}, 0) // !!false = false
	runOne(t, src, "not_eq", []uint64{3, 3}, 0)  // !(3==3) = false
	runOne(t, src, "not_eq", []uint64{3, 4}, 1)  // !(3==4) = true
}

func TestEndToEnd_Increment(t *testing.T) {
	src := `
        export fn count_up(): int {
            let i: int = 0;
            i++;
            i++;
            i++;
            return i;    // 3
        }
        export fn count_in_loop(n: i64): i64 {
            let i: i64 = 0;
            while i < n { i++; }
            return i;    // n
        }
    `
	runOne(t, src, "count_up", nil, 3)
	runOne(t, src, "count_in_loop", []uint64{10}, 10)
}

func TestEndToEnd_Decrement(t *testing.T) {
	src := `
        export fn count_down(n: i64): i64 {
            let i: i64 = n;
            while i > 0 { i--; }
            return i;    // 0
        }
    `
	runOne(t, src, "count_down", []uint64{5}, 0)
}

func TestEndToEnd_StringIndex(t *testing.T) {
	src := `
        export fn first_byte(): int { return "hello"[0]; }   // 'h' = 104
        export fn last_byte():  int { return "hello"[4]; }   // 'o' = 111
        export fn from_var(): int {
            let s: string = "hello";
            return s[1];  // 'e' = 101
        }
        export fn sum_bytes(): int {
            let s: string = "AB";  // 'A'=65, 'B'=66
            return s[0] + s[1];  // 131
        }
    `
	runOne(t, src, "first_byte", nil, 104)
	runOne(t, src, "last_byte", nil, 111)
	runOne(t, src, "from_var", nil, 101)
	runOne(t, src, "sum_bytes", nil, 131)
}

func TestEndToEnd_Power(t *testing.T) {
	src := `
        export fn square(x: int): int { return x ** 2; }
        export fn cube(x: int): int { return x ** 3; }
        export fn pow_zero(x: int): int { return x ** 0; }
        export fn right_assoc(): int { return 2 ** 3 ** 2; }
    `
	runOne(t, src, "square", []uint64{4}, 16)
	runOne(t, src, "cube", []uint64{3}, 27)
	runOne(t, src, "pow_zero", []uint64{999}, 1)
	runOne(t, src, "right_assoc", nil, 512) // 2 ** (3**2) = 2**9
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

	exportedFn := mod.ExportedFunction(fn)
	got, err := exportedFn.Call(ctx, args...)
	if err != nil {
		t.Fatalf("call: %v", err)
	}

	// wazero reuses the args slice for results. For i32 returns it only writes
	// the low 32 bits, leaving the high 32 bits = whatever was in args[0].
	// Mask when the declared result type is i32.
	results := exportedFn.Definition().ResultTypes()
	if len(results) > 0 && results[0] == api.ValueTypeI32 {
		got[0] = uint64(uint32(got[0]))
	}

	if got[0] != want {
		t.Fatalf("%s%v = %d, want %d", fn, args, got[0], want)
	}
}
