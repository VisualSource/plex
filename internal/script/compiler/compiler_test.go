package compiler_test

import (
	"context"
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/script"
	"github.com/VisualSource/plex/internal/script/compiler"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func TestCompilerFunctionCall(t *testing.T) {
	src := `
		fn add(a: float, b: float): float { return a + b; }
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

func TestBools(t *testing.T) {
	src := `
		fn getTrue(): bool {
			return true;
		}

		fn getTrue(): bool {
			return false;
		}
	`

	validateFromSnapshot(t, src, "bool_test.wat")
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

	validateFromSnapshot(t, src, "struct.wat")
}

func TestAlloc(t *testing.T) {
	src := `
		import heapPtr from "plex:globals";

		fn alloc(size: i32): i32 {
			let ptr = heapPtr;
			heapPtr = heapPtr + size;
			return ptr;
		}
	`

	validateFromSnapshot(t, src, "alloc.wat")
}

func TestArray(t *testing.T) {
	src := `
	fn main(){
		let a: int[] = [1,2];

		let b: int = 1;

		a[b];
	}
	`
	validateFromSnapshot(t, src, "array.wat")
}

func TestOperators(t *testing.T) {
	src := `
		fn max(a: int, b: int): int {
			if a > b { return a; }
			return b;
		}

		fn clamp(x: int, lo: int, hi: int): int {
			if x < lo { return lo; }
			if x > hi { return hi; }
			return x;
		}
		`

	validateByRun(t, src, []testRun{
		{
			fn:   "max",
			args: []uint64{1, 2},
			want: []uint64{2},
		},
		{
			fn:   "clamp",
			args: []uint64{5, 0, 4},
			want: []uint64{4},
		},
	}, func(ctx context.Context, r wazero.Runtime) {})
}

func TestBreak(t *testing.T) {
	src := `
	fn find_first(limit: int): int {
		let i = 0;
		while i < limit {
			if i == 5 {
				break;
			}
			i = i + 1;
		}
		return i;
	}
	`

	validateByRun(t, src, []testRun{
		{
			fn:   "find_first",
			args: []uint64{10},
			want: []uint64{5},
		},
	}, func(ctx context.Context, r wazero.Runtime) {})
}

func TestTernary(t *testing.T) {
	src := `
	fn abs(x: int): int {
		return x > 0 ? x : 0 - x;
	}
	`

	validateByRun(t, src, []testRun{
		{
			fn:   "abs",
			args: []uint64{api.EncodeI64(-7)},
			want: []uint64{7},
		},
		{
			fn:   "abs",
			args: []uint64{3},
			want: []uint64{3},
		},
	}, func(ctx context.Context, r wazero.Runtime) {})
}

func TestStringLen(t *testing.T) {
	src := `
		fn test(): int {
			let s = "Hello";
			return s.len();	
		}
	`

	validateByRun(t, src, []testRun{
		{
			fn:   "test",
			args: []uint64{},
			want: []uint64{5},
		},
	}, func(ctx context.Context, r wazero.Runtime) {})
}

func TestArrayLen(t *testing.T) {
	src := `
		fn test(): int {
			let s: int[] = [1,2];
			return s.len();	
		}
	`

	validateByRun(t, src, []testRun{
		{
			fn:   "test",
			args: []uint64{},
			want: []uint64{2},
		},
	}, func(ctx context.Context, r wazero.Runtime) {})
}

func TestArrayAppend(t *testing.T) {
	src := `
		fn test(): int {
			let s: int[] = [1,2];

			s.append(3);

			return s.len();	
		}
	`

	validateByRun(t, src, []testRun{
		{
			fn:   "test",
			args: []uint64{},
			want: []uint64{3},
		},
	}, func(ctx context.Context, r wazero.Runtime) {})
}

func TestArrayRemove(t *testing.T) {
	src := `
		fn test(): bool {
			let s: int[] = [10,20,30];

			let removed = s.remove(1);

			return removed == 20 && s.len() == 2;
		}
	`

	validateByRun(t, src, []testRun{
		{
			fn:   "test",
			args: []uint64{},
			want: []uint64{1},
		},
	}, func(ctx context.Context, r wazero.Runtime) {})
}

func TestCast(t *testing.T) {
	src := `
		fn main(): f64 {
			let n: i64 = 7;
			return (n as f64) / 2.0;
		}
	`

	validateByRun(t, src, []testRun{
		{
			fn:   "main",
			args: []uint64{},
			want: []uint64{api.EncodeF64(3.5)},
		},
	}, func(ctx context.Context, r wazero.Runtime) {})
}
