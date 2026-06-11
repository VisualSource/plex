package script_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/script"
)

func TestOperatorPrecedence(t *testing.T) {
	tokens := getTokens("1 + 2 * 4")

	parser := script.NewParser(tokens)

	node, err := parser.Parse()
	if err != nil {
		t.Fatal(err)
	}

	result, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", result)
}

func TestPostfixCall(t *testing.T) {
	tokens := getTokens("someFunction(1,2)")

	parser := script.NewParser(tokens)

	node, err := parser.Parse()
	if err != nil {
		t.Fatal(err)
	}

	result, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", result)
}

func TestPostfixMethodAccess(t *testing.T) {
	tokens := getTokens("someObject.helloWorld")

	parser := script.NewParser(tokens)

	node, err := parser.Parse()
	if err != nil {
		t.Fatal(err)
	}

	result, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", result)
}

func getTokens(input string) []script.Token {
	tok := script.NewTokenizer(strings.NewReader(input))

	tokens, err := tok.Tokenize()
	if err != nil {
		panic(err)
	}

	return tokens
}
