package html

import (
	"unicode/utf8"
)

const (
	Token_EOF = iota
	Token_Character
	Token_StartTag
	Token_Comment
	Token_EndTag
	Token_DOCTYPE
)

// https://html.spec.whatwg.org/#tokenization
type Token interface {
	GetType() int
	IsType(int) bool
}

type DoctypeToken struct {
	Name             *string
	PublicIdentifier *string
	SystemIdentifer  *string
	ForceQuirks      bool
}

func NewDOCTYPEToken() *DoctypeToken {
	return &DoctypeToken{
		ForceQuirks: false,
	}
}

func (d *DoctypeToken) GetType() int {
	return Token_DOCTYPE
}

func (d *DoctypeToken) IsType(t int) bool {
	return Token_DOCTYPE == t
}

type TagToken struct {
	Name        string
	Attrs       map[string]string
	Selfclosing *bool

	tagType int
	cavalue string
	caname  string
}

func NewStartToken() *TagToken {
	return &TagToken{
		tagType: Token_StartTag,
		Attrs:   make(map[string]string),
	}
}
func NewEndToken() *TagToken {
	return &TagToken{
		tagType: Token_EndTag,
		Attrs:   make(map[string]string),
	}
}

func (t TagToken) IsStartTag() bool {
	return t.tagType == Token_StartTag
}

func (t TagToken) IsEndTag() bool {
	return !t.IsStartTag()
}

func (t TagToken) GetType() int {
	return t.tagType
}

func (t TagToken) IsType(v int) bool {
	return t.tagType == v
}

func (t *TagToken) AppendStringToAttrName(value string) {
	t.caname += value
}
func (t *TagToken) AppendStringAttrValue(value string) {
	t.cavalue += value
}

// should do a token type check before calling
func (t *TagToken) FinishAttr() {
	if t.caname != "" {
		_, ok := t.Attrs[t.caname]
		if !ok {
			t.Attrs[t.caname] = t.cavalue
		}
	}
}

func (t *TagToken) NewAttr(name string, value string) {
	t.FinishAttr()
	t.caname = name
	t.cavalue = value
}

type TokenEOF struct{}

func (e *TokenEOF) GetType() int {
	return Token_EOF
}
func (e *TokenEOF) IsType(v int) bool {
	return v == Token_DOCTYPE
}

func NewEOFToken() *TokenEOF {
	return &TokenEOF{}
}

type TokenCharacter struct {
	Data rune
}

func (c *TokenCharacter) GetType() int {
	return Token_Character
}

func (c *TokenCharacter) IsType(v int) bool {
	return Token_Character == v
}

func NewCharacterToken(value rune) *TokenCharacter {
	return &TokenCharacter{
		Data: utf8.RuneError,
	}
}
func NewReplacementToken() *TokenCharacter {
	return &TokenCharacter{
		Data: utf8.RuneError,
	}
}

type CommentToken struct {
	Data string
}

func NewCommentToken() *CommentToken {
	return &CommentToken{}
}

func (c *CommentToken) GetType() int {
	return Token_Comment
}

func (c *CommentToken) IsType(value int) bool {
	return Token_Comment == value
}
