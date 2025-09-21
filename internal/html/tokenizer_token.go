package html

import (
	"errors"
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
type Token struct {
	token       int
	doctypeData *doctypeData
	tagData     *tagData
	data        string
}

type doctypeData struct {
	name             *string
	publicIdentifier *string
	systemIdentifier *string
	forceQuirks      bool
}

type tagData struct {
	name        string
	attrs       map[string]string
	selfClosing *bool
	cavalue     string
	caname      string
}

// #region DOCTYPE
func NewDOCTYPEToken() *Token {
	return &Token{
		token:       Token_DOCTYPE,
		doctypeData: &doctypeData{},
	}
}
func (t *Token) DOCTYPE_GetName() (*string, error) {
	if t.token != Token_DOCTYPE {
		return nil, errors.New("token is not a of type 'DOCTYPE'")
	}

	return t.doctypeData.name, nil
}
func (t *Token) DOCTYPE_GetPublicIdentifer() (*string, error) {
	if t.token != Token_DOCTYPE {
		return nil, errors.New("token is not a of type 'DOCTYPE'")
	}

	return t.doctypeData.publicIdentifier, nil
}
func (t *Token) DOCTYPE_GetSystemIdentifer() (*string, error) {
	if t.token != Token_DOCTYPE {
		return nil, errors.New("token is not a of type 'DOCTYPE'")
	}

	return t.doctypeData.systemIdentifier, nil
}
func (t *Token) DOCTYPE_GetForceQuirks() (bool, error) {
	if t.token != Token_DOCTYPE {
		return false, errors.New("token is not a of type 'DOCTYPE'")
	}
	return t.doctypeData.forceQuirks, nil
}
func (t *Token) DOCTYPE_SetName(value string) error {
	if t.token != Token_DOCTYPE {
		return errors.New("token is not a of type 'DOCTYPE'")
	}

	t.doctypeData.name = &value

	return nil
}
func (t *Token) DOCTYPE_SetPublicIdentifer(value string) error {
	if t.token != Token_DOCTYPE {
		return errors.New("token is not a of type 'DOCTYPE'")
	}

	t.doctypeData.publicIdentifier = &value

	return nil
}
func (t *Token) DOCTYPE_SetSystemIdentifer(value string) error {
	if t.token != Token_DOCTYPE {
		return errors.New("token is not a of type 'DOCTYPE'")
	}

	t.doctypeData.systemIdentifier = &value

	return nil
}
func (t *Token) DOCTYPE_SetForceQuirks(value bool) error {
	if t.token != Token_DOCTYPE {
		return errors.New("token is not a of type 'DOCTYPE'")
	}
	t.doctypeData.forceQuirks = value
	return nil
}
func (t *Token) DOCTYPE_AppendStringToName(value string) error {
	if t.token != Token_DOCTYPE {
		return errors.New("token is not a of type 'DOCTYPE'")
	}

	if t.doctypeData.name != nil {
		*t.doctypeData.name += value
	}

	return nil
}
func (t *Token) DOCTYPE_AppendStringToPublicIdent(value string) error {
	if t.token != Token_DOCTYPE {
		return errors.New("token is not a of type 'DOCTYPE'")
	}

	if t.doctypeData.publicIdentifier != nil {
		*t.doctypeData.publicIdentifier += value
	}

	return nil
}
func (t *Token) DOCTYPE_AppendStringToSystemIdent(value string) error {
	if t.token != Token_DOCTYPE {
		return errors.New("token is not a of type 'DOCTYPE'")
	}

	if t.doctypeData.systemIdentifier != nil {
		*t.doctypeData.systemIdentifier += value
	}

	return nil
}

//#endregion

//#region Tag

func NewTagToken(tokenType int) *Token {
	return &Token{
		token: tokenType,
		tagData: &tagData{
			attrs: make(map[string]string),
		},
	}
}

func NewStartTagToken() *Token {
	return NewTagToken(Token_StartTag)
}

func NewEndTagToken() *Token {
	return NewTagToken(Token_EndTag)
}

func (t *Token) Tag_GetName() (string, error) {
	if t.token != Token_StartTag && t.token != Token_EndTag {
		return "", errors.New("token is not a of type 'StartTag' or 'EndTag'")
	}

	return t.tagData.name, nil
}
func (t *Token) Tag_GetSelfClosing() (*bool, error) {
	if t.token != Token_StartTag && t.token != Token_EndTag {
		return nil, errors.New("token is not a of type 'StartTag' or 'EndTag'")
	}
	return t.tagData.selfClosing, nil
}

func (t *Token) Tag_SetName(value string) error {
	if t.token != Token_StartTag && t.token != Token_EndTag {
		return errors.New("token is not a of type 'StartTag' or 'EndTag'")
	}

	t.tagData.name = value
	return nil
}
func (t *Token) Tag_SetSelfClosing(value bool) error {
	if t.token != Token_StartTag && t.token != Token_EndTag {
		return errors.New("token is not a of type 'StartTag' or 'EndTag'")
	}

	t.tagData.selfClosing = &value
	return nil
}
func (t *Token) Tag_SetAttr(attr string, value string) error {
	if t.token != Token_StartTag && t.token != Token_EndTag {
		return errors.New("token is not a of type 'StartTag' or 'EndTag'")
	}

	t.tagData.attrs[attr] = value
	return nil
}

func (t *Token) Tag_AppendString_ToCurrentAttrName(value string) error {
	if t.token != Token_StartTag && t.token != Token_EndTag {
		return errors.New("token is not a of type 'StartTag' or 'EndTag'")
	}

	t.tagData.caname += value

	return nil
}
func (t *Token) Tag_AppendString_ToCurrentAttrValue(value string) error {
	if t.token != Token_StartTag && t.token != Token_EndTag {
		return errors.New("token is not a of type 'StartTag' or 'EndTag'")
	}

	t.tagData.cavalue += value

	return nil
}

// should do a token type check before calling
func (t *Token) Tag_FinishAttr() {
	if t.tagData.caname != "" {
		_, ok := t.tagData.attrs[t.tagData.caname]
		if !ok {
			t.tagData.attrs[t.tagData.caname] = t.tagData.cavalue
		}
	}
}

func (t *Token) Tag_NewAttr(name string, value string) error {
	if t.token != Token_StartTag && t.token != Token_EndTag {
		return errors.New("token is not a of type 'StartTag' or 'EndTag'")
	}

	t.Tag_FinishAttr()
	t.tagData.caname = name
	t.tagData.cavalue = value

	return nil
}

func (t *Token) Tag_AppendStringToName(value string) error {
	if t.token != Token_StartTag && t.token != Token_EndTag {
		return errors.New("token is not a of type 'StartTag' or 'EndTag'")
	}

	t.tagData.name += value
	return nil
}

func (t *Token) Tag_IsSelfClosingSet() (bool, error) {
	if t.token != Token_StartTag && t.token != Token_EndTag {
		return false, errors.New("token is not a of type 'StartTag' or 'EndTag'")
	}

	return t.tagData.selfClosing != nil, nil
}

//#endregion

func NewEOFToken() *Token {
	return &Token{token: Token_EOF}
}

func NewReplacementToken() *Token {
	return &Token{
		token: Token_Character,
		data:  string(utf8.RuneError),
	}
}

func NewCommentToken() *Token {
	return &Token{token: Token_Comment, data: ""}
}

func (t *Token) Comment_AppendString(value string) error {
	if t.token != Token_Comment {
		return errors.New("token is not of type 'Comment'")
	}

	t.data += value

	return nil
}

func NewCharToken(value rune) *Token {
	return &Token{
		token: Token_Character,
		data:  string(value),
	}
}
