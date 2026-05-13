package tokenizer_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/css/tokenizer"
)

type testCase struct {
	Source string  `json:"source"`
	Tokens []token `json:"tokens"`
}

func TestCssParser(t *testing.T) {
	files := loadTestFiles(t)

	for filename, tests := range files {
		for i, tt := range tests {
			testname := fmt.Sprintf("[%s]: test %d", filename, i)
			t.Run(testname, func(t *testing.T) {
				parser := tokenizer.NewCssTokenizer(strings.NewReader(tt.Source))

				tokens := make([]tokenizer.Token, 0)
				for {
					token, err := parser.ConsumeToken()
					if err != nil {
						t.Fatalf("parser returned error of %v", err)
						break
					}

					if token.IsToken() == tokenizer.TokenId_EOF {
						break
					}
					tokens = append(tokens, token)
				}

				validateTest(t, tokens, tt)
			})
		}
	}
}

type token struct {
	Type       string         `json:"type"`
	Raw        string         `json:"raw"`
	StartIndex int            `json:"startIndex"`
	EndIndex   int            `json:"endIndex"`
	Structured map[string]any `json:"structured"`
}

func loadTestFiles(t *testing.T) map[string][]testCase {
	t.Helper()

	values, _ := os.LookupEnv("TEST_CSS_TOKENIZER_SUBSET")
	allowed := strings.Split(values, ",")
	useSubset := values != "" && len(allowed) != 0

	if useSubset {
		t.Logf("using test subtest: %s", values)
	}

	paths, err := filepath.Glob(filepath.Join("testdata", "*.test"))
	if err != nil {
		t.Fatal(err)
	}

	tests := make(map[string][]testCase)

	for _, path := range paths {
		_, filename := filepath.Split(path)
		testname := filename[:len(filename)-len(filepath.Ext(path))]

		if useSubset && !slices.Contains(allowed, testname) {
			t.Logf("ignore test file '%s'", testname)
			continue
		}

		t.Logf("Using test file '%s'", testname)

		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		var contents struct {
			Tests []testCase `json:"tests"`
		}
		err = json.Unmarshal(source, &contents)
		if err != nil {
			t.Fatal(err)
		}

		tests[testname] = contents.Tests
	}

	return tests
}

func validateTest(t *testing.T, result []tokenizer.Token, tt testCase) {

}

func isToken(value string, id tokenizer.TokenId) bool {
	switch value {
	case "}-token":
		return id == tokenizer.TokenId_BracketCurlyClose
	case "{-token":
		return id == tokenizer.TokenId_BracketCurlyOpen
	case "[-token":
		return id == tokenizer.TokenId_BracketSquareOpen
	case "]-token":
		return id == tokenizer.TokenId_BracketSquareClose
	case "at-keyword-token":
		return id == tokenizer.TokenId_AtKeyword
	case "whitespace-token":
		return id == tokenizer.TokenId_Whitespace
	case "delim-token":
		return id == tokenizer.TokenId_Delim
	case "ident-token":
		return id == tokenizer.TokenId_Ident
	case "function-token":
		return id == tokenizer.TokenId_Function
	case "hash-token":
		return id == tokenizer.TokenId_Hash
	case "string-token":
		return id == tokenizer.TokenId_String
	case "bad-string-token":
		return id == tokenizer.TokenId_BadString
	case "url-token":
		return id == tokenizer.TokenId_Url
	case "bad-url-token":
		return id == tokenizer.TokenId_BadUrl
	case "number-token":
		return id == tokenizer.TokenId_Number
	case "percentage-token":
		return id == tokenizer.TokenId_Percentage
	case "dimension-token":
		return id == tokenizer.TokenId_Dimension
	case "cdo-token":
		return id == tokenizer.TokenId_CDO
	case "cdc-token":
		return id == tokenizer.TokenId_CDC
	case "colon-token":
		return id == tokenizer.TokenId_Colon
	case "semicolon":
		return id == tokenizer.TokenId_Semicolon
	case "comma":
		return id == tokenizer.TokenId_Comma
	case "(-token":
		return id == tokenizer.TokenId_BracketParamOpen
	case ")-token":
		return id == tokenizer.TokenId_BracketParamClose
	default:
		return false
	}
}
