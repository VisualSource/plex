package html_tokenizer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/utils"
	"github.com/kr/pretty"
)

type testCase struct {
	Description   string   `json:"description"`
	Input         string   `json:"input"`
	Output        [][]any  `json:"output"`
	InitialStates []string `json:"initialStates"`
	LastStartTag  string   `json:"lastStartTag"`
	DoubleEscaped bool     `json:"doubleEscaped"`
	Errors        []struct {
		Code string `json:"code"`
		Line int    `json:"line"`
		Col  int    `json:"col"`
	} `json:"errors"`
}

func TestTokenizer(t *testing.T) {
	files := loadTestCases(t)

	for filename, tests := range files {
		for _, tt := range tests {
			if len(tt.InitialStates) != 0 {
				for _, state := range tt.InitialStates {
					iState := resolveStateStrToType(state)

					testname := fmt.Sprintf("[%s](%s): %s", filename, state, tt.Description)
					input := strings.NewReader(resolveEncoding(tt.Input, tt.DoubleEscaped))
					t.Run(testname, func(t *testing.T) {
						tok := NewTokenizer(input)
						tok.SetState(iState)

						if tt.LastStartTag != "" {
							tok.lastStartTag.Set(tt.LastStartTag)
						}

						validate(t, tok, &tt)
					})

				}
				continue
			}

			testname := fmt.Sprintf("[%s]: %s", filename, tt.Description)
			input := strings.NewReader(resolveEncoding(tt.Input, tt.DoubleEscaped))
			t.Run(testname, func(t *testing.T) {
				tok := NewTokenizer(input)

				if tt.LastStartTag != "" {
					tok.lastStartTag.Set(tt.LastStartTag)
				}

				validate(t, tok, &tt)
			})
		}
	}
}

func loadTestCases(t *testing.T) map[string][]testCase {
	t.Helper()

	values, _ := os.LookupEnv("TEST_HTML_TOKENIZER_SUBSET")
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

func resolveStateStrToType(value string) TokenizerState {
	switch value {
	case "PLAINTEXT state":
		return State_PlainText
	case "RCDATA state":
		return State_RCData
	case "RAWTEXT state":
		return State_RawText
	case "Script data state":
		return State_ScriptData
	case "CDATA section state":
		return state_CDATA_Section
	default:
		return State_Data
	}
}

// UnescapeUnicode converts escaped unicode sequences (including double-escaped)
// into their actual unicode characters.
// e.g. "\\u00e9" -> "é", "\u00e9" -> "é"
func UnescapeUnicode(s string) (string, error) {
	// strconv.Unquote requires a quoted string, so wrap it.
	// First, normalize double-escaped sequences: \\uXXXX -> \uXXXX
	normalized := strings.ReplaceAll(s, `\\u`, `\u`)
	normalized = strings.ReplaceAll(normalized, `\\U`, `\U`)
	normalized = strings.ReplaceAll(normalized, `"`, `\"`)

	// Wrap in quotes so strconv.Unquote can parse it
	quoted := `"` + normalized + `"`
	unquoted, err := strconv.Unquote(quoted)
	if err != nil {
		return "", fmt.Errorf("failed to unquote: %w", err)
	}

	return unquoted, nil
}

func resolveEncoding(input string, isDoubleEscaped bool) string {
	if !isDoubleEscaped {
		return input
	}

	r, err := UnescapeUnicode(input)
	if err != nil {
		panic(err)
	}

	return r
}

func testOptStr(expected any, value utils.Option[string], name string, isDoubleEscaped bool, t *testing.T) {
	if expected == nil {
		if value.IsSome() {
			t.Fatalf("was expecting DOCTYPE prop %s to be nil but got: '%s'", name, *value.Value)
		}
		return
	}

	exValue := expected.(string)

	if value.IsNone() {
		t.Fatalf("DOCTYPE prop %s was expected to contain value '%s' but is nil", name, expected)
	}

	if resolveEncoding(exValue, isDoubleEscaped) != *value.Value {
		t.Fatalf("was expecting prop %s to have value '%s' but got '%s'", name, exValue, *value.Value)
	}
}

func validate(t *testing.T, tok *Tokenizer, tt *testCase) {
	t.Helper()

	iters := 0
	for {
		err := tok.Next()
		if err != nil {
			if err != io.EOF {
				t.Fatalf("reader error: %v", err)
			}
			break
		}
		iters++

		if iters > 1000 {
			t.Fatal("max iteration")
			break
		}
	}

	if len(tt.Output) != 0 {
		// test case collapse character tokens so we can't check here if the output and tokens match
		if len(tok.tokens) == 0 && len(tt.Output) != 0 {
			t.Fatalf("tokens miss match: was expecting %v but got %+v", tt.Output, pretty.Sprint(tok.tokens))
		}
		i := 0
		for _, arg := range tt.Output {

			if i > len(tok.tokens) {
				t.Fatalf("failed to access token at idx '%d' tokens(%d)", i, len(tok.tokens))
				break
			}

			token := tok.tokens[i]
			switch arg[0].(string) {
			case "StartTag":
				if tag, ok := token.(*TokenTag); ok && tag.t == TokenStartTag {
					if resolveEncoding(arg[1].(string), tt.DoubleEscaped) != tag.name {
						t.Fatalf("was expecting a name of '%s' but was given '%s'", arg[1], tag.name)
					}

					for name, value := range arg[2].(map[string]any) {
						tag, hasTag := tag.attrs[resolveEncoding(name, tt.DoubleEscaped)]
						if !hasTag {
							t.Fatalf("was expecting to have attribute with a name of '%s' but not was found", name)
						}

						if resolveEncoding(value.(string), tt.DoubleEscaped) != tag {
							t.Fatalf("was expecting attribute '%s' to have value of '%s' but got '%s'", name, value, tag)
						}
					}

					if len(arg) == 4 && !tag.selfClosing.Is(arg[3].(bool)) {
						t.Fatalf("was expecting self closing flag to be '%#v' but was given '%#v'", arg[3], tag.selfClosing.Value)
					}
				} else {
					t.Fatalf("was expecting an start tag but got '%#v'", token)
				}
			case "EndTag":
				if tag, ok := token.(*TokenTag); ok && tag.t == TokenEndTag {
					if resolveEncoding(arg[1].(string), tt.DoubleEscaped) != tag.name {
						t.Fatalf("was expecting end tag to have tag name of '%s' not '%s'", arg[1], tag.name)
					}
				} else {
					t.Fatalf("was expecting an end tag but got '%#v'", token)
				}
			case "DOCTYPE":
				if tag, ok := token.(*TokenDOCTYPE); ok {
					testOptStr(arg[1], tag.name, "name", tt.DoubleEscaped, t)
					testOptStr(arg[2], tag.publicIdentifier, "public_id", tt.DoubleEscaped, t)
					testOptStr(arg[3], tag.systemIdentifier, "system_id", tt.DoubleEscaped, t)

					correctness := arg[4].(bool)

					if correctness == true && tag.forceQuirks != false {
						t.Fatalf("was expecting DOCTYPE forceQuirks flag to be 'false' but got '%#v'", tag.forceQuirks)
					} else if correctness == false && tag.forceQuirks != true {
						t.Fatalf("was expecting DOCTYPE forceQuirks flag to be 'true' but got '%#v'", tag.forceQuirks)
					}
				} else {
					t.Fatalf("was expecting an doctype tag but got '%s'", pretty.Sprint(token))
				}
			case "Character":
				for _, char := range resolveEncoding(arg[1].(string), tt.DoubleEscaped) {
					if i >= len(tok.tokens) {
						t.Fatalf("was expecting a character token but none are left. expected: %v given: %s", arg[1], pretty.Sprint(tok.tokens))
					}

					if tag, ok := tok.tokens[i].(*TokenCharacter); ok {

						if char != tag.Value {
							t.Fatalf("was expecting character token to have value of '%c' but got '%c'", char, tag.Value)
						}
					} else {
						t.Fatalf("was expecting a character token but found '%s'", pretty.Sprint(tok.tokens[i]))
					}
					i++
				}

				continue
			case "Comment":
				if tag, ok := token.(*TokenComment); ok {
					if resolveEncoding(arg[1].(string), tt.DoubleEscaped) != tag.Value {
						t.Fatalf("was expecting comment to have value '%s' not '%s'", arg[1], tag.Value)
					}
				} else {
					t.Fatalf("was expecting an comment token but got: '%#v'", token)
				}
			}

			i++
		}
	}

	if len(tt.Errors) != 0 {
		for _, err := range tt.Errors {
			if !slices.ContainsFunc(tok.errors, func(e TokenizerError) bool {
				return string(e.Reason) == err.Code
			}) {
				t.Errorf("was expecting to find tokenizer error '%s' but tokenizer did not return one", err.Code)
			}
		}
	}
}
