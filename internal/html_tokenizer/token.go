package html_tokenizer

import (
	"unicode/utf8"

	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/utils"
)

type AttributesMap map[string]dom.Attribute

func (p AttributesMap) Get(key string) utils.StringOption {
	attr, ok := p[key]
	if ok {
		return utils.Some(attr.Value)
	}
	return utils.None[string]()
}

// https://html.spec.whatwg.org/#tokenization
type Token interface{ isToken() }

type TokenEOF struct{}

func (TokenEOF) isToken() {}
func NewTokenEOF() Token  { return &TokenEOF{} }

type TokenCharacter struct{ Value rune }

func (TokenCharacter) isToken()          {}
func NewTokenCharacter(value rune) Token { return &TokenCharacter{Value: value} }

type TokenTagType uint8

const (
	TokenEndTag   TokenTagType = 0
	TokenStartTag TokenTagType = 1
)

type TokenTag struct {
	t           TokenTagType
	name        string
	Attributes  AttributesMap
	selfClosing utils.BoolOption
}

func (t TokenTag) GetType() TokenTagType {
	return t.t
}
func (TokenTag) isToken() {}
func NewTokenTag(
	name string,
	tokenType TokenTagType,
	selfClosing utils.BoolOption,
) *TokenTag {
	return &TokenTag{
		name:        name,
		Attributes:  make(AttributesMap),
		t:           tokenType,
		selfClosing: selfClosing,
	}
}
func (p TokenTag) IsSelfClosingSet() bool {
	return p.selfClosing.Is(true)
}
func (p TokenTag) GetName() string {
	return p.name
}
func (p *TokenTag) SetName(value string) {
	p.name = value
}

type TokenComment struct{ Value string }

func (TokenComment) isToken()            {}
func NewTokenComment(value string) Token { return &TokenComment{Value: value} }

type TokenDOCTYPE struct {
	name             utils.StringOption
	publicIdentifier utils.StringOption
	systemIdentifier utils.StringOption
	forceQuirks      bool
}

func (d TokenDOCTYPE) GetName() utils.StringOption {
	return d.name
}
func (d TokenDOCTYPE) GetPublicIdentifier() utils.StringOption {
	return d.publicIdentifier
}
func (d TokenDOCTYPE) GetSystemIdentifier() utils.StringOption {
	return d.systemIdentifier
}
func (d TokenDOCTYPE) GetForceQuirks() bool {
	return d.forceQuirks
}

func (TokenDOCTYPE) isToken() {}
func NewTokenDOCTYPE(
	name utils.StringOption,
	publicIdentifier utils.StringOption,
	systemIdentifier utils.StringOption,
	forceQuirks bool,
) Token {
	return &TokenDOCTYPE{
		name,
		publicIdentifier,
		systemIdentifier,
		forceQuirks,
	}
}

func NewReplacementToken() Token {
	return NewTokenCharacter(utf8.RuneError)
}
