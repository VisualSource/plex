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
	TokenType_AND
	TokenType_Number
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

type Position struct {
	Col int64
	Row int64
}

func NewPosition(col, row int64) Position {
	return Position{Col: col, Row: row}
}

type Token interface {
	IsToken() TokenType
	Range() (Position, Position)
}

type DataToken struct {
	Type       TokenType
	Start, End Position
}

func NewDataToken(token TokenType, start, end Position) *DataToken {
	return &DataToken{
		Type:  token,
		Start: start,
		End:   end,
	}
}

func (d DataToken) IsToken() TokenType {
	return d.Type
}
func (d DataToken) Range() (Position, Position) {
	return d.Start, d.End
}

type ValueToken struct {
	ttype TokenType
	Value string
	Start Position
	End   Position
}

func NewNumberToken(value string, start, end Position) *ValueToken {
	return &ValueToken{
		ttype: TokenType_Number,
		Start: start,
		End:   end,
		Value: value,
	}
}

func NewIdentToken(value string, start, end Position) *ValueToken {
	return &ValueToken{
		ttype: TokenType_Ident,
		Start: start,
		End:   end,
		Value: value,
	}
}

func NewKeywordToken(value string, start, end Position) *ValueToken {
	return &ValueToken{
		ttype: TokenType_Keyword,
		Value: value,
		Start: start,
		End:   end,
	}
}

func NewStringToken(value string, start, end Position) *ValueToken {
	return &ValueToken{
		ttype: TokenType_String,
		Value: value,
		Start: start,
		End:   end,
	}
}

func (s ValueToken) IsToken() TokenType {
	return s.ttype
}

func (s ValueToken) Range() (Position, Position) {
	return s.Start, s.End
}
