package script

type TokenType uint

const (
	TokenType_EOF TokenType = iota
	TokenType_BracketCurlyOpen
	TokenType_BracketCulryClose
	TokenType_BracketSquareOpen
	TokenType_BracketSquareClose
	TokenType_BracketParamOpen
	TokenType_BracketParamClose
	TokenType_Semicolon
	TokenType_Colon
	TokenType_Comma
	TokenType_Dot
	TokenType_Number
	TokenType_Float
	TokenType_Keyword
	TokenType_Plus
	TokenType_Minus
	TokenType_Star
	TokenType_Div
	TokenType_Mod
	TokenType_Char
	TokenType_String
	TokenType_Ident
	TokenType_Question
	TokenType_Equal
	TokenType_FatArrow
	TokenType_LessThen
	TokenType_GreaterThen
	TokenType_LessThenOrEqual
	TokenType_GreaterThenOrEqaul
	TokenType_Incrment
	TokenType_Decrement
	TokenType_NotEqual
	TokenType_EqualEqual
	TokenType_OR
	TOkenType_AND
)

var delimMap = map[rune]TokenType{
	'.':  TokenType_Dot,
	'%':  TokenType_Mod,
	'{':  TokenType_BracketCurlyOpen,
	'}':  TokenType_BracketCulryClose,
	'[':  TokenType_BracketSquareOpen,
	']':  TokenType_BracketSquareClose,
	'(':  TokenType_BracketParamOpen,
	')':  TokenType_BracketParamClose,
	'\\': TokenType_Div,
	'*':  TokenType_Star,
	':':  TokenType_Colon,
	',':  TokenType_Comma,
	';':  TokenType_Semicolon,
	'-':  TokenType_Minus,
	'+':  TokenType_Plus,
	'?':  TokenType_Question,
	'=':  TokenType_Equal,
	'>':  TokenType_GreaterThen,
	'<':  TokenType_LessThen,
}

type Token interface {
	IsToken() TokenType
	Range() (int64, int64)
}

type DataToken struct {
	Type TokenType
	row  int64
	col  int64
}

func NewDataToken(token TokenType, row int64, col int64) *DataToken {
	return &DataToken{
		Type: token,
		row:  row,
		col:  col,
	}
}

func (d DataToken) IsToken() TokenType {
	return d.Type
}
func (d DataToken) Range() (int64, int64) {
	return d.row, d.col
}

type ValueToken struct {
	ttype TokenType
	row   int64
	col   int64
	Value string
}

func NewIdentToken(value string, row int64, col int64) *ValueToken {
	return &ValueToken{
		ttype: TokenType_Ident,
		row:   row,
		col:   col,
		Value: value,
	}
}

func NewKeywordToken(value string, row int64, col int64) *ValueToken {
	return &ValueToken{
		ttype: TokenType_Keyword,
		Value: value,
		row:   row,
		col:   col,
	}
}

func NewStringToken(value string, row int64, col int64) *ValueToken {
	return &ValueToken{
		ttype: TokenType_String,
		Value: value,
		row:   row,
		col:   col,
	}
}

func (s ValueToken) IsToken() TokenType {
	return s.ttype
}

func (s ValueToken) Range() (int64, int64) {
	return s.row, s.col
}
