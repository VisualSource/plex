package tokenizer

//https://www.w3.org/TR/css-syntax-3/#tokenization
type Token interface {
	isToken()
}

type IdentToken struct {
	Value string
}

func (i IdentToken) isToken() {}

const (
	IdentType_Hash = iota
	IdentType_Ident
	IdentType_Function
	IdentType_AtKeyword
	IdentType_String
	IdentType_Url
)

// <ident-token>, <function-token>, <at-keyword-token>, <hash-token>, <string-token>, and <url-token>
type MultiCharacterToken struct {
	Type  uint
	Value string
	Flag  string
}

func (i MultiCharacterToken) isToken() {}

const (
	CharType_Delim = iota
	CharType_AsRune
)

// <colon-token>, <semicolon-token>, <comma-token>, <[-token>, <]-token>, <(-token>, <)-token>, <{-token>, and <}-token>. <delim-token>
type SingleCharacterToken struct {
	Value rune
	Type  uint
}

func (b SingleCharacterToken) isToken() {}

type EOFToken struct{}

func (b EOFToken) isToken() {}

// <number-token>, <percentage-token>, and <dimension-token>
type NumericToken struct {
	Value float64
	Unit  string
}

func (n NumericToken) isToken() {}

const (
	Info_Whitespace = iota
	Info_CDO
	Info_CDC
	Info_BadString
	Info_BadUrl
)

// <bad-string-token>, <bad-url-token>, <whitespace-token>, <CDO-token>, <CDC-token>,
type InfoToken struct {
	Type uint
}

func (i InfoToken) isToken() {}
