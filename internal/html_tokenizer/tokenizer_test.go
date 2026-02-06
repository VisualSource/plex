package html_tokenizer

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

type testCase struct {
	Description   string   `json:"description"`
	Input         string   `json:"input"`
	Output        [][]any  `json:"output"`
	InitialStates []string `json:"initialStates"`
	LastStartTag  string   `json:"lastStartTag"`
	DoubleEscaped bool     `json:"doubleEscaped"`
	Errors        []struct {
		Code string
		line int
		col  int
	} `json:"errors"`
}

// https://github.com/html5lib/html5lib-tests

func TestTokenizer(t *testing.T) {
	tests := []testCase{}
	for _, tt := range tests {

		if len(tt.InitialStates) != 0 {
			for _, state := range tt.InitialStates {
				iState := resolveStateStrToType(state)

				t.Run(fmt.Sprintf("%s: %s", state, tt.Description), func(t *testing.T) {
					tok := NewTokenizer(strings.NewReader(resolveInputEncoding(tt.Input, tt.DoubleEscaped)))
					tok.state = iState

					runTest(t, tok, &tt)
				})
			}
			continue
		}

		t.Run(tt.Description, func(t *testing.T) {
			tok := NewTokenizer(strings.NewReader(resolveInputEncoding(tt.Input, tt.DoubleEscaped)))
			runTest(t, tok, &tt)
		})
	}
}

func resolveStateStrToType(value string) TokenizerState {
	return State_Data
}

func resolveInputEncoding(input string, isDoubleEscaped bool) string {
	if isDoubleEscaped {
		return input
	}

	return input
}

func runTest(t *testing.T, tok *Tokenizer, tt *testCase) {
	t.Helper()
	for {
		err := tok.Next()
		if err != nil {
			if err != io.EOF {
				t.Fatalf("reader error: %v", err)
			}
			break
		}
	}

	if len(tt.Output) != 0 {

		for idx, arg := range tt.Output {
			if idx > len(tok.tokens) {
				t.Fatalf("token count does not match expecting output")
			}
			token := tok.tokens[idx]

			switch arg[0].(string) {
			case "StartTag":

			case "EndTag":
				if tag, ok := token.(*TokenEndTag); ok {
					if arg[1].(string) != tag.name {
						t.Fatalf("was expecting end tag to have tag name of '%s' not '%v'", arg[1], tag.name)
					}
				} else {
					t.Fatalf("was expecting an end tag but got '%v'", token)
				}
			case "DOCTYPE":
			case "Character":
			case "Comment":
				if tag, ok := token.(*TokenComment); ok {
					if arg[1].(string) != tag.Value {
						t.Fatalf("was expecting comment to have value '%s' not '%s'", arg[1], tag.Value)
					}

				} else {
					t.Fatalf("was expecting an comment token but got: '%v'", token)
				}

			}
		}
	}

	if len(tt.Errors) != 0 {
		// check errors
	}

}
