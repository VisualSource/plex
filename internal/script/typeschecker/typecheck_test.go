package typeschecker_test

import (
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/script"
	"github.com/VisualSource/plex/internal/script/typeschecker"
	"github.com/kr/pretty"
)

func TestCheck(t *testing.T) {
	output, err := script.Parse(strings.NewReader(`
		fn add(a: float, b: float): float { return a + b; }

		fn double(x: float): float {
			return add(x,x);
		}
	`))
	if err != nil {
		t.Fatal(err)
	}

	if err = typeschecker.Check(output); err != nil {
		t.Fatal(err)
	}

	pretty.Printf("%# v", output)
}

func TestUsingImportVars(t *testing.T) {
	src := `
	import heapPtr from "plex:globals";

	fn alloc(size: i32): i32 {
		let ptr = heapPtr;
		heapPtr = heapPtr + size;
		return ptr;
	}`

	output, err := script.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}

	if err = typeschecker.Check(output); err != nil {
		t.Fatal(err)
	}

	pretty.Printf("%# v", output)
}

func TestStructImpl(t *testing.T) {
	src := `
	struct Point {
		x: float;
		y: float;
	}

	impl Point {
		fn getX(): float {
			return self.x;
		}
		fn doubleX(): float { 
			return self.getX() * 2.0; 
		}
		/*fn distanceTo(other: Point): float {
			//Don't have power operator (should add one)
			return ((self.x - other.x) *  (self.x - other.x)) + ((self.y - other.y) * (self.y - other.y));
		}*/
	}

	fn main(){
		let point = Point(1,1);

		point.getX();
	}
	`

	output, err := script.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}

	if err = typeschecker.Check(output); err != nil {
		t.Fatal(err)
	}

	pretty.Printf("%# v", output)
}

func TestArray(t *testing.T) {
	src := `
	import print from "plex:console";

		fn main(){
			let a: int[] = [1,2];

			let b: int = 1;

			a[b];
		}
	`

	output, err := script.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}

	if err = typeschecker.Check(output); err != nil {
		t.Fatal(err)
	}

	pretty.Printf("%# v", output)

}
