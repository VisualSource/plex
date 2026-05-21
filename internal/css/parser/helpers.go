package parser

import (
	"strings"

	"github.com/VisualSource/plex/internal/css/tokenizer"
)

func isDelim(token tokenizer.Token, value rune) bool {
	if token.IsToken() != tokenizer.TokenId_Delim {
		return false
	}

	iv, ok := token.(*tokenizer.SingleCharacterToken)
	if !ok {
		return false
	}

	return iv.Value == value
}

func isIdent(token tokenizer.Token, value string, insensitive bool) bool {
	if token.IsToken() != tokenizer.TokenId_Ident {
		return false
	}

	iv, ok := token.(*tokenizer.MultiCharacterToken)
	if !ok {
		return false
	}
	if insensitive {
		return strings.EqualFold(iv.Value, value)
	}

	return iv.Value == value
}

func isCustomPropertyName(e tokenizer.Token) bool {
	if ident, ok := e.(*tokenizer.MultiCharacterToken); ok {
		return strings.HasPrefix(ident.Value, "--")
	}

	return false
}

func isNotWhitespace(e tokenizer.Token) bool {
	return e.IsToken() != tokenizer.TokenId_Whitespace
}
func isSimpleBlockWithCurlyOpen(e tokenizer.Token) bool {
	if block, ok := e.(*SimpleBlock); ok {
		return block.StartDelim.IsToken() == tokenizer.TokenId_BracketCurlyOpen
	}
	return false
}

var bracketMap map[tokenizer.TokenId]tokenizer.TokenId = map[tokenizer.TokenId]tokenizer.TokenId{
	tokenizer.TokenId_BracketCurlyOpen:  tokenizer.TokenId_BracketCurlyClose,
	tokenizer.TokenId_BracketParamOpen:  tokenizer.TokenId_BracketParamClose,
	tokenizer.TokenId_BracketSquareOpen: tokenizer.TokenId_BracketSquareClose,
}

func getTokenValueAsString(v tokenizer.Token) string {
	if v, ok := v.(tokenizer.MultiCharacterToken); ok {
		return v.Value
	}
	return ""
}
