package parser

import (
	"strings"

	"github.com/VisualSource/plex/internal/css/tokenizer"
)

func IsDelim(token tokenizer.Token, value rune) bool {
	if token.IsToken() != tokenizer.TokenId_Delim {
		return false
	}

	iv, ok := token.(*tokenizer.SingleCharacterToken)
	if !ok {
		return false
	}

	return iv.Value == value
}

func IsIdent(token tokenizer.Token, value string, insensitive bool) bool {
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

// preludeLooksLikeCustomPropertyDecl reports whether the first two
// non-whitespace prelude values are an ident-token whose value starts with
// "--" followed by a colon-token. Used by consumeQualifiedRule to recognise
// preludes that would otherwise be misparsed as a custom-property declaration.
func preludeLooksLikeCustomPropertyDecl(prelude []tokenizer.Token) bool {
	var first, second tokenizer.Token
	for _, v := range prelude {
		if v.IsToken() == tokenizer.TokenId_Whitespace {
			continue
		}
		if first == nil {
			first = v
			continue
		}
		second = v
		break
	}
	if first == nil || second == nil {
		return false
	}
	return first.IsToken() == tokenizer.TokenId_Ident &&
		isCustomPropertyName(first) &&
		second.IsToken() == tokenizer.TokenId_Colon
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

func GetTokenValueAsString(v tokenizer.Token) string {
	if v, ok := v.(*tokenizer.MultiCharacterToken); ok {
		return v.Value
	}
	return ""
}

// valueSourceSegment returns the substring of the post-preprocessing source
// that spans decl.Value's tokens. Returns "" if decl.Value is empty.
func (p *CssParser) valueSourceSegment(decl *Declaration) string {
	if len(decl.Value) == 0 {
		return ""
	}
	start, _ := decl.Value[0].Range()
	_, end := decl.Value[len(decl.Value)-1].Range()
	return p.tok.Source(start, end)
}
