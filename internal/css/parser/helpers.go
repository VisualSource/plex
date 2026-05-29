package css_parser

import (
	"strings"

	css_tokenizer "github.com/VisualSource/plex/internal/css/tokenizer"
)

func IsDelim(token css_tokenizer.Token, value rune) bool {
	if token == nil || token.IsToken() != css_tokenizer.TokenId_Delim {
		return false
	}

	iv, ok := token.(*css_tokenizer.SingleCharacterToken)
	if !ok {
		return false
	}

	return iv.Value == value
}

func IsIdent(token css_tokenizer.Token, value string, insensitive bool) bool {
	if token == nil || token.IsToken() != css_tokenizer.TokenId_Ident {
		return false
	}

	iv, ok := token.(*css_tokenizer.MultiCharacterToken)
	if !ok {
		return false
	}
	if insensitive {
		return strings.EqualFold(iv.Value, value)
	}

	return iv.Value == value
}

func isCustomPropertyName(e css_tokenizer.Token) bool {
	if ident, ok := e.(*css_tokenizer.MultiCharacterToken); ok {
		return strings.HasPrefix(ident.Value, "--")
	}

	return false
}

func isNotWhitespace(e css_tokenizer.Token) bool {
	return e.IsToken() != css_tokenizer.TokenId_Whitespace
}

// preludeLooksLikeCustomPropertyDecl reports whether the first two
// non-whitespace prelude values are an ident-token whose value starts with
// "--" followed by a colon-token. Used by consumeQualifiedRule to recognise
// preludes that would otherwise be misparsed as a custom-property declaration.
func preludeLooksLikeCustomPropertyDecl(prelude []css_tokenizer.Token) bool {
	var first, second css_tokenizer.Token
	for _, v := range prelude {
		if v.IsToken() == css_tokenizer.TokenId_Whitespace {
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
	return first.IsToken() == css_tokenizer.TokenId_Ident &&
		isCustomPropertyName(first) &&
		second.IsToken() == css_tokenizer.TokenId_Colon
}
func isSimpleBlockWithCurlyOpen(e css_tokenizer.Token) bool {
	if block, ok := e.(*SimpleBlock); ok {
		return block.StartDelim.IsToken() == css_tokenizer.TokenId_BracketCurlyOpen
	}
	return false
}

var bracketMap map[css_tokenizer.TokenId]css_tokenizer.TokenId = map[css_tokenizer.TokenId]css_tokenizer.TokenId{
	css_tokenizer.TokenId_BracketCurlyOpen:  css_tokenizer.TokenId_BracketCurlyClose,
	css_tokenizer.TokenId_BracketParamOpen:  css_tokenizer.TokenId_BracketParamClose,
	css_tokenizer.TokenId_BracketSquareOpen: css_tokenizer.TokenId_BracketSquareClose,
}

func GetTokenValueAsString(v css_tokenizer.Token) string {
	if v, ok := v.(*css_tokenizer.MultiCharacterToken); ok {
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
