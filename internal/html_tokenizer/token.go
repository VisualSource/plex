package html_tokenizer

import (
	"unicode/utf8"

	"github.com/VisualSource/plex/internal/utils"
)

type AttributesMap map[string]string

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
	attrs       AttributesMap
	selfClosing utils.BoolOption
}

func (TokenTag) isToken() {}
func NewTokenTag(
	name string,
	tokenType TokenTagType,
	selfClosing utils.BoolOption,
) Token {
	return &TokenTag{
		name:        name,
		attrs:       make(AttributesMap),
		t:           tokenType,
		selfClosing: selfClosing,
	}
}

func (p TokenTag) GetName() string {
	return p.name
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
