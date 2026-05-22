package parser_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/css/tokenizer"
)

func Test_ParseComponentValue(t *testing.T) {
	testCases := loadTestFile(t, "on-component-value.test")

	for i, testCase := range testCases {
		name := fmt.Sprintf("test %d", i)

		t.Run(name, func(t *testing.T) {
			parser := parser.NewCssParser()

			result, err := parser.ParseComponentValue(strings.NewReader(testCase.input))
			if err != nil {
				t.Fatalf("failed to parse value: %s", err)
			}

			validateTest(t, testCase, []tokenizer.Token{result})
		})
	}
}

//#region helpers

type testCase struct {
	input string
	ast   []node
}

type node struct {
	nodeType string
	args     []any
}

func validateTest(t *testing.T, testCase testCase, result []tokenizer.Token) {
	t.Helper()

}

func loadTestFile(t *testing.T, file string) []testCase {
	t.Helper()

	path := filepath.Join("testdata", file)
	source, err := os.ReadFile(path)
	if err != nil {
		panic("failed to open test file")
	}

	raw := make([]any, 0)

	if err = json.Unmarshal(source, &raw); err != nil {
		panic("failed to unmarshal test file content")
	}

	testCases := make([]testCase, 0)

	for chunk := range slices.Chunk(raw, 2) {
		input := chunk[0].(string)

		ast := make([]node, 0)

		for _, item := range chunk[1].([][]any) {
			t := item[0].(string)

			n := node{
				nodeType: t,
				args:     item[1:],
			}

			ast = append(ast, n)
		}

		tc := testCase{
			input,
			ast,
		}

		testCases = append(testCases, tc)
	}

	return testCases
}

//#endregion
