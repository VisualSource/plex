package html_tokenizer

import (
	"strings"
	"testing"
)

func TestTokenizer_StateData(t *testing.T) {
	r := strings.NewReader("<div data-name='test'><p>Hello World</p></div>")

	tokenizer := NewTokenizer(r)

}
