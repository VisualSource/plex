package html_tokenizer

import (
	"io"
	"strings"
	"testing"
)

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

			for i, v := range tt.check {

				switch i {
				case "tokens":

				case "state":
					if to.state != v.(int) {
						t.Errorf("state_Data() failed: expected: '%v' but got '%v'", v, to.state)
					}
				case "rstate":
					if to.rstate.IsNone() {
						t.Error("rstate was not set with a value")
					}

					rstate := to.rstate.MustSome()

					if rstate != v.(int) {
						t.Errorf("state_Data() failed: expected: '%v' but got '%v'", v, to.state)
					}
				default:
					t.Errorf("no check setup for '%v'", i)
				}
			}
		})
	}
}
