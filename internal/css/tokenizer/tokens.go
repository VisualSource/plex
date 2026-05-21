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
//
// Range returns the rune offsets of the token in the post-preprocessing
// input stream, with End being exclusive (i.e. End - Start equals the rune
// count of the token's source span). For composite types built by the
// parser, Range spans the underlying tokens of the construct.
type Token interface {
	IsToken() TokenId
	Range() (start, end int)
}

// rangeSetter is the unexported counterpart used by the tokenizer to stamp
// positions onto a token after it has been constructed.
type rangeSetter interface {
	setRange(start, end int)
}

// <ident-token>, <function-token>, <at-keyword-token>, <hash-token>, <string-token>, and <url-token>
type MultiCharacterToken struct {
	Type  TokenId
	Value string
	Flag  string
	Start int
	End   int
}

func (i MultiCharacterToken) IsToken() TokenId       { return i.Type }
func (i MultiCharacterToken) Range() (int, int)      { return i.Start, i.End }
func (i *MultiCharacterToken) setRange(s, e int)     { i.Start, i.End = s, e }

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
	Start int
	End   int
}

func (b SingleCharacterToken) IsToken() TokenId       { return b.Type }
func (b SingleCharacterToken) Range() (int, int)      { return b.Start, b.End }
func (b *SingleCharacterToken) setRange(s, e int)     { b.Start, b.End = s, e }

func NewSingleCharacterToken(t TokenId, value rune) *SingleCharacterToken {
	return &SingleCharacterToken{
		Type:  t,
		Value: value,
	}
}

type EOFToken struct {
	Start int
	End   int
}

func (b EOFToken) IsToken() TokenId       { return TokenId_EOF }
func (b EOFToken) Range() (int, int)      { return b.Start, b.End }
func (b *EOFToken) setRange(s, e int)     { b.Start, b.End = s, e }

func NewEOFToken() *EOFToken { return &EOFToken{} }

// <number-token>, <percentage-token>, and <dimension-token>
type NumericToken struct {
	Type TokenId

	Value float64

	Unit string // px, and other stuff

	Flag string // integer, number
	Sign rune   // '+','-' or missing (0)

	Start int
	End   int
}

func (n NumericToken) IsToken() TokenId       { return n.Type }
func (n NumericToken) Range() (int, int)      { return n.Start, n.End }
func (n *NumericToken) setRange(s, e int)     { n.Start, n.End = s, e }

func NewNumericToken(t TokenId, value float64, sign rune) *NumericToken {
	return &NumericToken{
		Type:  t,
		Value: value,
		Sign:  sign,
	}
}

// <colon-token>, <semicolon-token>, <comma-token>, <[-token>, <]-token>, <(-token>, <)-token>, <{-token>, and <}-token>. <bad-string-token>, <bad-url-token>, <whitespace-token>, <CDO-token>, <CDC-token>,
type DataToken struct {
	Type  TokenId
	Start int
	End   int
}

func (i DataToken) IsToken() TokenId       { return i.Type }
func (i DataToken) Range() (int, int)      { return i.Start, i.End }
func (i *DataToken) setRange(s, e int)     { i.Start, i.End = s, e }

func NewDataToken(t TokenId) *DataToken { return &DataToken{Type: t} }

type UnicodeRangeToken struct {
	Start int // first code point in the range (NOT a source offset)
	End   int // last code point in the range (NOT a source offset)

	SrcStart int // rune offset of the token's first rune in the source
	SrcEnd   int // rune offset just past the token's last rune in the source
}

func (i UnicodeRangeToken) IsToken() TokenId       { return TokenId_UnicodeRange }
func (i UnicodeRangeToken) Range() (int, int)      { return i.SrcStart, i.SrcEnd }
func (i *UnicodeRangeToken) setRange(s, e int)     { i.SrcStart, i.SrcEnd = s, e }

func NewUnicodeRangeToken(start int, end int) *UnicodeRangeToken {
	return &UnicodeRangeToken{
		Start: start,
		End:   end,
	}
}
