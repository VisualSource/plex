package tokenizer_test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/css/tokenizer"
	"github.com/kr/pretty"
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

	for i, expected := range tt.Tokens {
		if len(result) <= i {
			t.Fatal(pretty.Sprintf("was expecting %v results but was given %v", tt.Tokens, result))
			break
		}

		if !isToken(expected.Type, result[i].IsToken()) {
			t.Fatalf("was expecting token type of %s but was given %s at idx %d", expected.Type, tokenAsText(result[i].IsToken()), i)
		}

		switch result[i].IsToken() {
		case tokenizer.TokenId_AtKeyword:
			if tok, ok := (result[i].(*tokenizer.MultiCharacterToken)); ok {
				exp := expected.Structured["value"].(string)
				if tok.Value != exp {
					t.Fatalf("was expecting a value of '%s' but was given '%s'", exp, tok.Value)
				}
			} else {
				t.Fatalf("current token is not a multi character token got %#v", result[i])
			}
		case tokenizer.TokenId_Delim:
			if tok, ok := (result[i].(*tokenizer.SingleCharacterToken)); ok {
				exp := []rune(expected.Structured["value"].(string))[0]

				if tok.Value != exp {
					t.Fatalf("was expecting a value of '%c' but was given '%c'", exp, tok.Value)
				}
			} else {
				t.Fatalf("current token is not a single character token got %#v", result[i])
			}
		case tokenizer.TokenId_Dimension:
			if tok, ok := (result[i].(*tokenizer.NumericToken)); ok {
				v := expected.Structured["value"].(float64)
				ttv := expected.Structured["type"].(string)
				u := expected.Structured["unit"].(string)

				if !(math.Abs(tok.Value-v) < 1e-9 && ttv == tok.Flag && u == tok.Unit) {
					t.Fatalf("was expecting Value(%f) Type(%s) Unit(%s) but was given Value(%f) Type(%s) Unit(%s)", v, ttv, u, tok.Value, tok.Flag, tok.Unit)
				}
			} else {
				t.Fatalf("current token is not a multi character token got %#v", result[i])
			}
		case tokenizer.TokenId_Function:
			if tok, ok := (result[i].(*tokenizer.MultiCharacterToken)); ok {
				exp := expected.Structured["value"].(string)
				if tok.Value != exp {
					t.Fatalf("was expecting a value of '%s' but was given '%s'", exp, tok.Value)
				}
			} else {
				t.Fatalf("current token is not a multi character token got %#v", result[i])
			}
		case tokenizer.TokenId_Hash:
			if tok, ok := (result[i].(*tokenizer.MultiCharacterToken)); ok {
				exp := expected.Structured["value"].(string)
				id := expected.Structured["type"].(string)
				if !(tok.Value == exp && tok.Flag == id) {
					t.Fatalf("was expecting a value of '%s','%s' but was given '%s','%s'", exp, id, tok.Value, tok.Flag)
				}
			} else {
				t.Fatalf("current token is not a multi character token got %#v", result[i])
			}
		case tokenizer.TokenId_Number:
			if tok, ok := (result[i].(*tokenizer.NumericToken)); ok {
				v := expected.Structured["value"].(float64)
				ttv := expected.Structured["type"].(string)

				if !(math.Abs(tok.Value-v) < 1e-9 && ttv == tok.Flag) {
					t.Fatalf("was expecting Value(%f) Type(%s) but was given Value(%f) Type(%s)", v, ttv, tok.Value, tok.Flag)
				}
			} else {
				t.Fatalf("current token is not a multi character token got %#v", result[i])
			}
		case tokenizer.TokenId_String:
			if tok, ok := (result[i].(*tokenizer.MultiCharacterToken)); ok {
				exp := expected.Structured["value"].(string)
				if tok.Value != exp {
					t.Fatalf("was expecting a value of '%s' but was given '%s'", exp, tok.Value)
				}
			} else {
				t.Fatalf("current token is not a multi character token got %#v", result[i])
			}
		case tokenizer.TokenId_Url:
			if tok, ok := (result[i].(*tokenizer.MultiCharacterToken)); ok {
				exp := expected.Structured["value"].(string)
				if tok.Value != exp {
					t.Fatalf("was expecting a value of '%s' but was given '%s'", exp, tok.Value)
				}
			} else {
				t.Fatalf("current token is not a multi character token got %#v", result[i])
			}
		}
	}

}

func tokenAsText(id tokenizer.TokenId) string {
	switch id {
	case tokenizer.TokenId_BracketCurlyClose:
		return "}-token"
	case tokenizer.TokenId_BracketCurlyOpen:
		return "{-token"
	case tokenizer.TokenId_BracketSquareOpen:
		return "[-token"
	case tokenizer.TokenId_BracketSquareClose:
		return "]-token"
	case tokenizer.TokenId_AtKeyword:
		return "at-keyword-token"
	case tokenizer.TokenId_Whitespace:
		return "whitespace-token"
	case tokenizer.TokenId_Delim:
		return "delim-token"
	case tokenizer.TokenId_Ident:
		return "ident-token"
	case tokenizer.TokenId_Function:
		return "function-token"
	case tokenizer.TokenId_Hash:
		return "hash-token"
	case tokenizer.TokenId_String:
		return "string-token"
	case tokenizer.TokenId_BadString:
		return "bad-string-token"
	case tokenizer.TokenId_Url:
		return "url-token"
	case tokenizer.TokenId_BadUrl:
		return "bad-url-token"
	case tokenizer.TokenId_Number:
		return "number-token"
	case tokenizer.TokenId_Percentage:
		return "percentage-token"
	case tokenizer.TokenId_Dimension:
		return "dimension-token"
	case tokenizer.TokenId_CDO:
		return "cdo-token"
	case tokenizer.TokenId_CDC:
		return "cdc-token"
	case tokenizer.TokenId_Colon:
		return "colon-token"
	case tokenizer.TokenId_Semicolon:
		return "semicolon"
	case tokenizer.TokenId_Comma:
		return "comma"
	case tokenizer.TokenId_BracketParamOpen:
		return "(-token"
	case tokenizer.TokenId_BracketParamClose:
		return ")-token"
	default:
		return "UNKNOWN"
	}

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
	case "CDO-token":
		return id == tokenizer.TokenId_CDO
	case "CDC-token":
		return id == tokenizer.TokenId_CDC
	case "colon-token":
		return id == tokenizer.TokenId_Colon
	case "semicolon-token":
		return id == tokenizer.TokenId_Semicolon
	case "comma-token":
		return id == tokenizer.TokenId_Comma
	case "(-token":
		return id == tokenizer.TokenId_BracketParamOpen
	case ")-token":
		return id == tokenizer.TokenId_BracketParamClose
	default:
		return false
	}
}
