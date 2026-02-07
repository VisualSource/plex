package html_tokenizer

import (
	"unicode/utf8"

	"github.com/MadAppGang/dingo/pkg/dgo"
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

type TokenStartTag struct {
	name        string
	attrs       AttributesMap
	selfClosing dgo.Option[bool]
	cavalue     string
	caname      string
}

func (TokenStartTag) isToken() {}
func NewTokenStartTag(name string, attrs AttributesMap, selfClosing dgo.Option[bool], cavalue string, caname string) Token {
	return &TokenStartTag{name: name, attrs: attrs, selfClosing: selfClosing, cavalue: cavalue, caname: caname}
}

type TokenComment struct{ Value string }

func (TokenComment) isToken()            {}
func NewTokenComment(value string) Token { return &TokenComment{Value: value} }

type TokenEndTag struct{ name string }

func (TokenEndTag) isToken()           {}
func NewTokenEndTag(name string) Token { return &TokenEndTag{name: name} }

type TokenDOCTYPE struct {
	name             dgo.Option[string]
	publicIdentifier dgo.Option[string]
	systemIdentifier dgo.Option[string]
	forceQuirks      bool
}

func (TokenDOCTYPE) isToken() {}
func NewTokenDOCTYPE(name dgo.Option[string], publicIdentifier dgo.Option[string], systemIdentifier dgo.Option[string], forceQuirks bool) Token {
	return &TokenDOCTYPE{name: name, publicIdentifier: publicIdentifier, systemIdentifier: systemIdentifier, forceQuirks: forceQuirks}
}

func NewReplacementToken() Token {
	return NewTokenCharacter(utf8.RuneError)
}
