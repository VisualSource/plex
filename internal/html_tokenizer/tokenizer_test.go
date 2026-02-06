package html_tokenizer

import (
	"io"
	"slices"
	"strings"
	"testing"
)

func handleCompare(t *testing.T, checks map[string]any, to *Tokenizer) {
	for i, v := range checks {

		switch i {
		case "tokens":
			result := slices.EqualFunc(v.([]Token), to.tokens, func(e1, e2 Token) bool {
				if target, ok := e1.(TokenCharacter); ok {
					current, ok := e2.(TokenCharacter)
					if !ok {
						return false
					}
					return target.Value == current.Value
				}

				if _, ok := e1.(TokenEOF); ok {
					_, ok := e2.(TokenEOF)
					if !ok {
						return false
					}
					return true
				}

				if target, ok := e1.(TokenComment); ok {
					current, ok := e2.(TokenComment)
					if !ok {
						return false
					}
					return target.Value == current.Value
				}

				return false
			})

			if !result {
				t.Errorf("failed: expected: '%v' to match '%v'", to.tokens, v)
			}
		case "state":
			if to.state != v.(TokenizerState) {
				t.Errorf("failed: expected: '%v' but got '%v'", v, to.state)
			}
		case "rstate":
			if to.rstate != v.(TokenizerState) {
				t.Errorf("failed: expected: '%v' but got '%v'", v, to.state)
			}
		default:
			t.Errorf("no check setup for '%v'", i)
		}
	}

}

func TestTokenizer_state_Data(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		stream  io.RuneReader
		wantErr bool
		check   map[string]any
	}{
		{
			name:    "Test Ampersand",
			stream:  strings.NewReader("&"),
			wantErr: false,
			check: map[string]any{
				"rstate": State_Data,
				"state":  state_CharacterReference,
			},
		},
		{
			name:    "Test Less Than",
			stream:  strings.NewReader("<"),
			wantErr: false,
			check: map[string]any{
				"state": state_TagOpen,
			},
		},
		{
			name:    "Test EOF",
			stream:  strings.NewReader(""),
			wantErr: false,
			check: map[string]any{
				"tokens": []Token{NewTokenEOF()},
			},
		},
		{
			name:    "Test Anything Else",
			stream:  strings.NewReader("anything"),
			wantErr: false,
			check: map[string]any{
				"tokens": []Token{NewTokenCharacter('a')},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			to := NewTokenizer(tt.stream)
			gotErr := to.state_Data()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("state_Data() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("state_Data() succeeded unexpectedly")
			}

			handleCompare(t, tt.check, to)
		})
	}
}

func TestTokenizer_state_RCData(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		stream  io.RuneReader
		wantErr bool
		check   map[string]any
	}{
		{
			name:    "Ampersand",
			stream:  strings.NewReader("&"),
			wantErr: false,
			check: map[string]any{
				"rstate": State_RCData,
				"state":  state_CharacterReference,
			},
		},
		{
			name:    "Less Than Sign",
			stream:  strings.NewReader("<"),
			wantErr: false,
			check: map[string]any{
				"state": state_RCData_LessThanSign,
			},
		},
		{
			name:    "Null",
			stream:  strings.NewReader("\u0000"),
			wantErr: false,
			check: map[string]any{
				"tokens": []Token{NewReplacementToken()},
			},
		},
		{
			name:   "Anything",
			stream: strings.NewReader("anything"),
			check: map[string]any{
				"tokens": []Token{NewTokenCharacter('a')},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			to := NewTokenizer(tt.stream)
			gotErr := to.state_RCData()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("state_RCData() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("state_RCData() succeeded unexpectedly")
			}

			handleCompare(t, tt.check, to)
		})
	}
}
