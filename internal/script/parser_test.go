package script_test

import (
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/script"
)

func TestParseExpression(t *testing.T) {
	tokens := getTokens("1 + 2 * 4")

	parser := script.NewParser(tokens)

	node, err := parser.Parse()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%# v", node)
}

func getTokens(input string) []script.Token {
	tok := script.NewTokenizer(strings.NewReader(input))

	tokens, err := tok.Tokenize()
	if err != nil {
		panic(err)
	}

	return tokens
}
