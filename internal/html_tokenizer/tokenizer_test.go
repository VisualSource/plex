package html_tokenizer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"

	"github.com/MadAppGang/dingo/pkg/dgo"
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
							tok.lastStartTag = dgo.Some(tt.LastStartTag)
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
					tok.lastStartTag = dgo.Some(tt.LastStartTag)
				}

				validate(t, tok, &tt)
			})
		}
	}
}

func loadTestCases(t *testing.T) map[string][]testCase {
	t.Helper()

	values, isSet := os.LookupEnv("TEST_HTML_TOKENIZER_SUBSET")
	allowed := strings.Split(values, ",")
	useSubset := len(allowed) != 0

	if isSet && !useSubset {
		t.Fatalf("no subset of tests where set!")
	}

	t.Logf("using test subtest: %s", values)

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

func resolveEncoding(input string, isDoubleEscaped bool) string {
	if isDoubleEscaped {
		return norm.NFC.String(input)
	}

	return input
}

func testOptStr(expected any, value dgo.Option[string], name string, isDoubleEscaped bool, t *testing.T) {
	if expected == nil {
		if value.Some != nil {
			t.Fatalf("was expecting DOCTYPE prop %s to be nil but got: '%s'", name, *value.Some)
		}
		return
	}

	exValue := expected.(string)

	if value.IsNone() {
		t.Fatalf("DOCTYPE prop %s was expected to contain value '%s' but is nil", name, expected)
	}

	if resolveEncoding(exValue, isDoubleEscaped) != *value.Some {
		t.Fatalf("was expecting prop %s to have value '%s' but got '%s'", name, exValue, *value.Some)
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
		if len(tok.tokens) == 0 && len(tt.Output) != 0 {
			t.Fatalf("missing tokens: was expecting %v", tt.Output)
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

					if len(arg) == 4 && arg[3].(bool) != *tag.selfClosing.Some {
						t.Fatalf("was expecting self closing flag to be '%#v' but was given '%#v'", arg[3], tag.selfClosing.Some)
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
					t.Fatalf("was expecting an doctype tag but got '%v'", token)
				}
			case "Character":
				for _, char := range resolveEncoding(arg[1].(string), tt.DoubleEscaped) {
					if tag, ok := tok.tokens[i].(*TokenCharacter); ok {

						if char != tag.Value {
							t.Fatalf("was expecting character token to have value of '%c' but got '%c'", char, tag.Value)
						}
					} else {
						t.Fatalf("was expecting a character token but found '%#v'", tok.tokens[i])
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
