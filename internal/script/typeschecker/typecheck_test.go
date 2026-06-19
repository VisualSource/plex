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
