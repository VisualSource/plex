package parser

import (
	"github.com/VisualSource/plex/internal/css/tokenizer"
	"github.com/VisualSource/plex/internal/utils"
)

const (
	TokenId_SimpleBlock     tokenizer.TokenId = 25
	TokenId_DeclarationList tokenizer.TokenId = 26
	TokenId_RuleList        tokenizer.TokenId = 27
	TokenId_Function        tokenizer.TokenId = 28
	TokenId_Declaration     tokenizer.TokenId = 29
	TokenId_Rule            tokenizer.TokenId = 30
)

type Stylesheet struct {
	Value    []*Rule
	Location utils.StringOption
}

type Rule struct {
	Name    tokenizer.Token
	Prelude []tokenizer.Token
	Value   tokenizer.Token
}

func (r Rule) IsToken() tokenizer.TokenId {
	return TokenId_Rule
}

type SimpleBlock struct {
	StartDelim tokenizer.Token
	Value      []tokenizer.Token
}

func (s SimpleBlock) IsToken() tokenizer.TokenId {
	return TokenId_SimpleBlock
}

type Declaration struct {
	Name      tokenizer.Token
	Value     []tokenizer.Token
	Important bool
}

func (d Declaration) IsToken() tokenizer.TokenId {
	return TokenId_Declaration
}

type DeclarationList struct{}

func (d DeclarationList) IsToken() tokenizer.TokenId {
	return TokenId_DeclarationList
}

type RuleList struct{}

func (r RuleList) IsToken() tokenizer.TokenId {
	return TokenId_RuleList
}

type Function struct {
	Name  string
	Value []tokenizer.Token
}

func (f Function) IsToken() tokenizer.TokenId {
	return TokenId_Function
}
