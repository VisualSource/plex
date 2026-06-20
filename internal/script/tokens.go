package script

import "fmt"

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
	TokenType_Power
)

func (t TokenType) String() string {
	switch t {
	case TokenType_BracketCurlyOpen:
		return "{"
	case TokenType_BracketCulryClose:
		return "}"
	case TokenType_BracketSquareOpen:
		return "["
	case TokenType_BracketSquareClose:
		return "]"
	case TokenType_BracketParamOpen:
		return "("
	case TokenType_BracketParamClose:
		return ")"
	case TokenType_Semicolon:
		return ";"
	case TokenType_Colon:
		return ":"
	case TokenType_Comma:
		return ","
	case TokenType_Dot:
		return "."
	case TokenType_Keyword:
		return "#keyword"
	case TokenType_Plus:
		return "+"
	case TokenType_Minus:
		return "-"
	case TokenType_Star:
		return "*"
	case TokenType_Div:
		return "/"
	case TokenType_Mod:
		return "%"
	case TokenType_String:
		return "#string"
	case TokenType_Ident:
		return "#ident"
	case TokenType_Question:
		return "?"
	case TokenType_Equal:
		return "="
	case TokenType_FatArrow:
		return "=>"
	case TokenType_LessThen:
		return "<"
	case TokenType_GreaterThen:
		return ">"
	case TokenType_LessThenOrEqual:
		return "<="
	case TokenType_GreaterThenOrEqaul:
		return ">="
	case TokenType_Incrment:
		return "++"
	case TokenType_Decrement:
		return "--"
	case TokenType_NotEqual:
		return "!="
	case TokenType_EqualEqual:
		return "=="
	case TokenType_OR:
		return "||"
	case TokenType_AND:
		return "&&"
	case TokenType_Number:
		return "#number"
	case TokenType_Power:
		return "**"
	default:
		return fmt.Sprintf("TokenType(%d)", t)
	}
}

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

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Row, p.Col)
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
