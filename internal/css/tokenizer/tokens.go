package tokenizer

type TokenId uint

const (
	TokenId_Ident TokenId = iota
	TokenId_FunctionToken
	TokenId_AtKeyword
	TokenId_Hash
	TokenId_String
	TokenId_BadString
	TokenId_Url
	TokenId_BadUrl
	TokenId_Delim
	TokenId_Number
	TokenId_Percentage
	TokenId_Dimension
	TokenId_Whitespace
	TokenId_CDO
	TokenId_CDC
	TokenId_Colon
	TokenId_Semicolon
	TokenId_Comma
	TokenId_BracketSquareOpen
	TokenId_BracketSquareClose
	TokenId_BracketParamOpen
	TokenId_BracketParamClose
	TokenId_BracketCurlyOpen
	TokenId_BracketCurlyClose
	TokenId_EOF
	TokenId_UnicodeRange
)

// https://www.w3.org/TR/css-syntax-3/#tokenization
type Token interface {
	IsToken() TokenId
}

// <ident-token>, <function-token>, <at-keyword-token>, <hash-token>, <string-token>, and <url-token>
type MultiCharacterToken struct {
	Type  TokenId
	Value string
	Flag  string
}

func (i MultiCharacterToken) IsToken() TokenId {
	return i.Type
}
func NewMultiCharacterToken(t TokenId, value string) *MultiCharacterToken {
	return &MultiCharacterToken{
		Type:  t,
		Value: value,
	}
}

// <delim-token>
type SingleCharacterToken struct {
	Type  TokenId
	Value rune
}

func (b SingleCharacterToken) IsToken() TokenId {
	return b.Type
}

func NewSingleCharacterToken(t TokenId, value rune) *SingleCharacterToken {
	return &SingleCharacterToken{
		Type:  t,
		Value: value,
	}
}

type EOFToken struct{}

func (b EOFToken) IsToken() TokenId { return TokenId_EOF }
func NewEOFToken() *EOFToken        { return &EOFToken{} }

// <number-token>, <percentage-token>, and <dimension-token>
type NumericToken struct {
	Type TokenId

	Value float64

	Unit string // px, and other stuff

	Flag string // integer, number
	Sign rune   // '+','-' or missing (0)
}

func (n NumericToken) IsToken() TokenId { return n.Type }
func NewNumericToken(t TokenId, value float64, sign rune) *NumericToken {
	return &NumericToken{
		Type:  t,
		Value: value,
		Sign:  sign,
	}
}

// <colon-token>, <semicolon-token>, <comma-token>, <[-token>, <]-token>, <(-token>, <)-token>, <{-token>, and <}-token>. <bad-string-token>, <bad-url-token>, <whitespace-token>, <CDO-token>, <CDC-token>,
type DataToken struct {
	Type TokenId
}

func (i DataToken) IsToken() TokenId    { return i.Type }
func NewDataToken(t TokenId) *DataToken { return &DataToken{Type: t} }

type UnicodeRangeToken struct {
	Start int
	End   int
}

func (i UnicodeRangeToken) IsToken() TokenId { return TokenId_UnicodeRange }

func NewUnicodeRangeToken(start int, end int) *UnicodeRangeToken {
	return &UnicodeRangeToken{
		Start: start,
		End:   end,
	}
}
