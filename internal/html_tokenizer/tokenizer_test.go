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
	}{
		{
			name:    "test tokenize content",
			stream:  strings.NewReader("<div data-test='content'>Text Content</div>"),
			wantErr: false,
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
		})
	}
}
