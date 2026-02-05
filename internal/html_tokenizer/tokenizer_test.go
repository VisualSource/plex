package html_tokenizer

import (
	"io"
	"strings"
	"testing"
)

func TestTokenizer_Next(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		stream  io.RuneReader
		wantErr bool
	}{
		{
			name:    "",
			stream:  strings.NewReader(""),
			wantErr: false,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			to := NewTokenizer(tt.stream)
			gotErr := to.Next()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Next() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Next() succeeded unexpectedly")
			}
		})
	}
}

func TestTokenizer_ReconsumeToken(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		stream io.RuneReader
		// Named input parameters for target function.
		token Token
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			to := NewTokenizer(tt.stream)
			to.ReconsumeToken(tt.token)
		})
	}
}

func TestTokenizer_ConsumeToken(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		stream io.RuneReader
		want   Token
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			to := NewTokenizer(tt.stream)
			got := to.ConsumeToken()
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("ConsumeToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenizer_state_Data(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		stream  io.RuneReader
		wantErr bool
	}{
		// TODO: Add test cases.
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

func TestTokenizer_state_RCData(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		stream  io.RuneReader
		wantErr bool
	}{
		// TODO: Add test cases.
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
		})
	}
}

func TestTokenizer_state_RawText(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		stream  io.RuneReader
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			to := NewTokenizer(tt.stream)
			gotErr := to.state_RawText()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("state_RawText() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("state_RawText() succeeded unexpectedly")
			}
		})
	}
}

func TestTokenizer_state_ScriptData(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		stream  io.RuneReader
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			to := NewTokenizer(tt.stream)
			gotErr := to.state_ScriptData()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("state_ScriptData() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("state_ScriptData() succeeded unexpectedly")
			}
		})
	}
}

func TestTokenizer_state_PlainText(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		stream  io.RuneReader
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			to := NewTokenizer(tt.stream)
			gotErr := to.state_PlainText()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("state_PlainText() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("state_PlainText() succeeded unexpectedly")
			}
		})
	}
}
