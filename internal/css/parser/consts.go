package css_parser

import (
	css_tokenizer "github.com/VisualSource/plex/internal/css/tokenizer"
	"github.com/VisualSource/plex/internal/utils"
)

const (
	TokenId_SimpleBlock        css_tokenizer.TokenId = 26
	TokenId_DeclarationList    css_tokenizer.TokenId = 27
	TokenId_RuleList           css_tokenizer.TokenId = 28
	TokenId_Function           css_tokenizer.TokenId = 29
	TokenId_Declaration        css_tokenizer.TokenId = 30
	TokenId_Rule               css_tokenizer.TokenId = 31
	TokenId_NestedDeclarations css_tokenizer.TokenId = 32
)

type Stylesheet struct {
	Rules    []*Rule
	Location utils.StringOption
	Origin   int
}

type Rule struct {
	Name         string
	Prelude      []css_tokenizer.Token
	Declarations *DeclarationList
	ChildRules   []css_tokenizer.Token
	Start        int
	End          int
}

func (r Rule) IsToken() css_tokenizer.TokenId { return TokenId_Rule }
func (r Rule) Range() (int, int)              { return r.Start, r.End }

type SimpleBlock struct {
	StartDelim css_tokenizer.Token
	Value      []css_tokenizer.Token
	Start      int
	End        int
}

func (s SimpleBlock) IsToken() css_tokenizer.TokenId { return TokenId_SimpleBlock }
func (s SimpleBlock) Range() (int, int)              { return s.Start, s.End }

type Declaration struct {
	Name         css_tokenizer.Token
	Value        []css_tokenizer.Token
	Important    bool
	OriginalText utils.StringOption
	Start        int
	End          int
}

func (d Declaration) IsToken() css_tokenizer.TokenId { return TokenId_Declaration }
func (d Declaration) Range() (int, int)              { return d.Start, d.End }

type DeclarationList struct {
	Value []*Declaration
	Start int
	End   int
}

func (d DeclarationList) IsToken() css_tokenizer.TokenId { return TokenId_DeclarationList }
func (d DeclarationList) Range() (int, int)              { return d.Start, d.End }

type NestedDeclarations struct {
	Value *DeclarationList
	Start int
	End   int
}

func (n NestedDeclarations) IsToken() css_tokenizer.TokenId { return TokenId_NestedDeclarations }
func (n NestedDeclarations) Range() (int, int)              { return n.Start, n.End }

type RuleList struct {
	Start int
	End   int
}

func (r RuleList) IsToken() css_tokenizer.TokenId { return TokenId_RuleList }
func (r RuleList) Range() (int, int)              { return r.Start, r.End }

type Function struct {
	Name  string
	Value []css_tokenizer.Token
	Start int
	End   int
}

func (f Function) IsToken() css_tokenizer.TokenId { return TokenId_Function }
func (f Function) Range() (int, int)              { return f.Start, f.End }
