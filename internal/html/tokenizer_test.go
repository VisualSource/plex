package html_test

import (
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/html"
)

func makeDOCTYPE(name string) html.Token {
	token := *html.NewDOCTYPEToken()
	token.DOCTYPE_SetName(name)
	return token
}

func makeStartTag(name string, attrs map[string]string) html.Token {
	token := html.NewStartTagToken()
	token.Tag_SetName(name)

	for name, value := range attrs {
		token.Tag_SetAttr(name, value)
	}

	return *token
}

func makeEndTag(name string) html.Token {
	token := html.NewEndTagToken()
	token.Tag_SetName(name)

	return *token
}

func testEq(a, b []html.Token) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTokenizer_Parse(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		want    []html.Token
		wantErr bool
		input   string
	}{
		{
			name:    "Parse Doc",
			wantErr: false,
			want: []html.Token{
				makeDOCTYPE("html"),
				makeStartTag("html", map[string]string{
					"lang": "en",
				}),
				makeEndTag("html"),
				html.NewEOFToken(),
			},
			input: "<!DOCTYPE html><html lang=\"en\"></html>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataStream := strings.NewReader(tt.input)
			var to html.Tokenizer = html.NewTokenizer(dataStream)
			got, gotErr := to.Parse()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Parse() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Parse() succeeded unexpectedly")
			}
			if !testEq(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
