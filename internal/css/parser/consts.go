package parser

import (
	"github.com/VisualSource/plex/internal/css/tokenizer"
	"github.com/VisualSource/plex/internal/utils"
)

const (
	TokenId_SimpleBlock     tokenizer.TokenId = 26
	TokenId_DeclarationList tokenizer.TokenId = 27
	TokenId_RuleList        tokenizer.TokenId = 28
	TokenId_Function        tokenizer.TokenId = 29
	TokenId_Declaration     tokenizer.TokenId = 30
	TokenId_Rule            tokenizer.TokenId = 31
)

type Stylesheet struct {
	Value    []*Rule
	Location utils.StringOption
}

type Rule struct {
	Name       tokenizer.Token
	Prelude    []tokenizer.Token
	ChildRules []tokenizer.Token
	Start      int
	End        int
}

func (r Rule) IsToken() tokenizer.TokenId { return TokenId_Rule }
func (r Rule) Range() (int, int)          { return r.Start, r.End }

type SimpleBlock struct {
	StartDelim tokenizer.Token
	Value      []tokenizer.Token
	Start      int
	End        int
}

func (s SimpleBlock) IsToken() tokenizer.TokenId { return TokenId_SimpleBlock }
func (s SimpleBlock) Range() (int, int)          { return s.Start, s.End }

type Declaration struct {
	Name         tokenizer.Token
	Value        []tokenizer.Token
	Important    bool
	OriginalText utils.StringOption
	Start        int
	End          int
}

func (d Declaration) IsToken() tokenizer.TokenId { return TokenId_Declaration }
func (d Declaration) Range() (int, int)          { return d.Start, d.End }

type DeclarationList struct {
	Start int
	End   int
}

func (d DeclarationList) IsToken() tokenizer.TokenId { return TokenId_DeclarationList }
func (d DeclarationList) Range() (int, int)          { return d.Start, d.End }

type RuleList struct {
	Start int
	End   int
}

func (r RuleList) IsToken() tokenizer.TokenId { return TokenId_RuleList }
func (r RuleList) Range() (int, int)          { return r.Start, r.End }

type Function struct {
	Name  string
	Value []tokenizer.Token
	Start int
	End   int
}

func (f Function) IsToken() tokenizer.TokenId { return TokenId_Function }
func (f Function) Range() (int, int)          { return f.Start, f.End }
