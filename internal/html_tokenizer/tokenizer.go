package html_tokenizer

import (
	"io"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/MadAppGang/dingo/pkg/dgo"
	"github.com/ianlewis/runeio"
)

const (
	State_Data = iota
	State_RCData
	State_RawText
	State_ScriptData
	State_PlainText

	state_TagOpen
	state_EndTagOpen
	state_TagName

	state_RCData_LessThanSign
	state_RCData_EndTagOpen
	state_RCData_EndTagName

	state_RawText_LessThanSign
	state_RawText_EndTagOpen
	state_RawText_EndTagName

	state_ScriptData_LessThanSign
	state_ScriptData_EndTagOpen
	state_ScriptData_EndTagName
	state_ScriptData_EscapeStart
	state_ScriptData_EscapeStartDash

	state_ScriptData_Escaped
	state_ScriptData_EscapedDash
	state_ScriptData_EscapedDashDash
	state_ScriptData_EscapedLessThanSign
	state_ScriptData_EscapedEndTagOpen
	state_ScriptData_EscapedEndTagName

	state_ScriptData_DoubleEscapedStart
	state_ScriptData_DoubleEscaped
	state_ScriptData_DoubleEscapedDash
	state_ScriptData_DoubleEscapedDashDash
	state_ScriptData_DoubleEscapedLessThanSign
	state_ScriptData_DoubleEscapeEnd

	state_BeforeAttributeName
	state_AttributeName
	state_AfterAttributeName

	state_BeforeAttributeValue
	state_AttributValue_DoubleQuoted
	state_AttributValue_SingleQuoted
	state_AttributValue_Unquoted
	state_AfterAttributeValue_Quoted

	state_SelfClosingStartTag
	state_BogusComment
	state_MarkupDeclarationOpen

	state_CommentStart
	state_CommentStartDash
	state_Comment
	state_CommentLessThanSign
	state_CommentLessThanSignBang
	state_CommentLessThanSignBangDash
	state_CommentLessThanSignBangDashDash
	state_CommentEndDash
	state_CommentEnd
	state_CommentEndBang

	state_DOCTYPE
	state_BeforeDOCTYPEName
	state_DOCTYPE_Name
	state_AfterDOCTYPE_Name

	state_AfterDOCTYPE_PublicKeyword
	state_BeforeDOCTYPE_PublicIdentifier
	state_DOCTYPE_PublicIdentifier_DoubleQuoted
	state_DOCTYPE_PublicIdentifier_SingleQuoted
	state_AfterDOCTYPE_PublicIdentifier

	state_BetweenDOCTYPE_PublicAndSystemIdentifiers

	state_AfterDOCTYPE_SystemKeyword
	state_BeforeDOCTYPE_SystemIdentifer
	state_DOCTYPE_SystemIdentifier_DoubleQuoted
	state_DOCTYPE_SystemIdentifier_SingleQuoted
	state_AfterDOCTYPE_SystemIdentifier

	state_BogusDOCTYPE

	state_CDATA_Section
	state_CDATA_SectionBracket
	state_CDATA_SectionEnd

	state_CharacterReference
	state_NamedCharacterReference
	state_AmbiguousAmpersand

	state_NumericCharacterReference
	state_HexadecimalCharacterReferenceStart
	state_DecimalCharacterReferenceStart
	state_HexadecimalCharacterReference
	state_DecimalCharacterReference
	state_NumericCharacterReferenceEnd
)

// https://html.spec.whatwg.org/#tokenization
type Tokenizer struct {
	reader                 *runeio.RuneReader
	state                  int
	rstate                 dgo.Option[int]
	tokens                 []Token
	characterReferenceCode int
	tempbuffer             string
	workingToken           Token
	lastStartTag           dgo.Option[string]
}

func NewTokenizer(stream io.RuneReader) Tokenizer {
	reader := runeio.NewReader(stream)

	return Tokenizer{
		reader: reader,
		state:  State_Data,
	}
}

func (t *Tokenizer) Next() error {
	var err error = nil

	switch t.state {
	case State_Data:
		err = t.state_Data()
	case State_RCData:
		err = t.state_RCData()
	case State_RawText:
		err = t.state_RawText()
	case State_ScriptData:
		err = t.state_ScriptData()
	case State_PlainText:
		err = t.state_PlainText()

	case state_TagOpen:
		err = t.state_TagOpen()
	case state_EndTagOpen:
		err = t.state_EndTagOpen()
	case state_TagName:
		err = t.state_TagName()

	case state_RCData_LessThanSign:
		err = t.state_RCData_LessThen()
	case state_RCData_EndTagOpen:
		err = t.state_RCData_EndTagOpen()
	case state_RCData_EndTagName:
		err = t.state_RCData_EndTagName()
	case state_RawText_LessThanSign:
		err = t.state_RawText_LessThenSign()
	case state_RawText_EndTagOpen:
		err = t.state_RawText_EndTagOpen()
	case state_RawText_EndTagName:
		err = t.state_RawText_EndTagName()

	case state_ScriptData_LessThanSign:
		err = t.state_ScriptData_LessThanSign()
	case state_ScriptData_EndTagOpen:
		err = t.state_ScriptData_EndTagOpen()
	case state_ScriptData_EndTagName:
		err = t.state_ScriptData_EndTagName()
	case state_ScriptData_EscapeStart:
		err = t.state_ScriptData_Escape_Start()
	case state_ScriptData_EscapeStartDash:
		err = t.state_ScriptData_Escape_StartDash()

	case state_ScriptData_Escaped:
		err = t.state_ScriptData_Escaped()
	case state_ScriptData_EscapedDash:
		err = t.state_ScriptData_Escaped_Dash()
	case state_ScriptData_EscapedDashDash:
		err = t.state_ScriptData_Escaped_DashDash()
	case state_ScriptData_EscapedLessThanSign:
		err = t.state_ScriptData_Escaped_LessThanSign()
	case state_ScriptData_EscapedEndTagOpen:
		err = t.state_ScriptData_Escaped_EndTagOpen()
	case state_ScriptData_EscapedEndTagName:
		err = t.state_ScriptData_Escaped_EndTagName()

	case state_ScriptData_DoubleEscapedStart:
		err = t.state_ScriptData_DoubleEscapedStart()
	case state_ScriptData_DoubleEscaped:
		err = t.state_ScriptData_DoubleEscaped()
	case state_ScriptData_DoubleEscapedDash:
		err = t.state_ScriptData_DoubleEscapedDash()
	case state_ScriptData_DoubleEscapedDashDash:
		err = t.state_ScriptData_DoubleEscapedDashDash()
	case state_ScriptData_DoubleEscapedLessThanSign:
		err = t.state_ScriptData_DoubleEscaped_LessThanSign()
	case state_ScriptData_DoubleEscapeEnd:
		err = t.state_ScriptData_DoubleEscapedEnd()

	case state_BeforeAttributeName:
		err = t.state_BeforeAttributeName()
	case state_AttributeName:
		err = t.state_AttributeName()
	case state_AfterAttributeName:
		err = t.state_AfterAttributeName()

	case state_BeforeAttributeValue:
		err = t.state_BeforeAttributeValue()
	case state_AttributValue_DoubleQuoted:
		err = t.state_AttributeValue_DoubleQuote()
	case state_AttributValue_SingleQuoted:
		err = t.state_AttributeValue_SignleQuote()
	case state_AttributValue_Unquoted:
		err = t.state_AttributeValue_Unquoted()
	case state_AfterAttributeValue_Quoted:
		err = t.state_AfterAttributeValue_Quoted()
	case state_SelfClosingStartTag:
		err = t.state_SelfClosingStartTag()
	case state_BogusComment:
		err = t.state_BogusComment()
	case state_MarkupDeclarationOpen:
		err = t.state_MarkupDeclarationOpen()

	case state_CommentStart:
		err = t.state_CommentStart()
	case state_CommentStartDash:
		err = t.state_Comment_StartDash()
	case state_Comment:
		err = t.state_Comment()
	case state_CommentLessThanSign:
		err = t.state_Comment_LessThanSign()
	case state_CommentLessThanSignBang:
		err = t.state_Comment_LessThanSign_Bang()
	case state_CommentLessThanSignBangDash:
		err = t.state_Comment_LessThanSign_BangDash()
	case state_CommentLessThanSignBangDashDash:
		err = t.state_Comment_LessThanSign_BangDashDash()
	case state_CommentEndDash:
		err = t.state_Comment_EndDash()
	case state_CommentEnd:
		err = t.state_CommentEnd()
	case state_CommentEndBang:
		err = t.state_Comment_EndBang()

	case state_DOCTYPE:
		err = t.state_DOCTYPE()
	case state_BeforeDOCTYPEName:
		err = t.state_BeforeDOCTYPEName()
	case state_DOCTYPE_Name:
		err = t.state_DOCTYPE_Name()
	case state_AfterDOCTYPE_Name:
		err = t.state_AfterDOCTYPE_Name()

	case state_AfterDOCTYPE_PublicKeyword:
		err = t.state_AfterDOCTYPE_PublicKeyword()
	case state_BeforeDOCTYPE_PublicIdentifier:
		err = t.state_BeforeDOCTYPE_PublicIdentifier()
	case state_DOCTYPE_PublicIdentifier_DoubleQuoted:
		err = t.state_DOCTYPE_PublicIdentifier_DoubleQuoted()
	case state_DOCTYPE_PublicIdentifier_SingleQuoted:
		err = t.state_DOCKTYPE_PublicIdentifier_SingleQuoted()
	case state_AfterDOCTYPE_PublicIdentifier:
		err = t.state_AfterDOCTYPE_PublicIdentifier()

	case state_BetweenDOCTYPE_PublicAndSystemIdentifiers:
		err = t.state_BetweenDOCTYPE_PublicAndSystemIdent()

	case state_AfterDOCTYPE_SystemKeyword:
		err = t.state_AfterDOCTYPE_SystemKeyword()
	case state_BeforeDOCTYPE_SystemIdentifer:
		err = t.state_BeforeDOCTYPE_SystemIdentifier()
	case state_DOCTYPE_SystemIdentifier_DoubleQuoted:
		err = t.state_DOCTYPE_SystemIdentifier_DoubleQuoted()
	case state_DOCTYPE_SystemIdentifier_SingleQuoted:
		err = t.state_DOCTYPE_SystemIdentifier_SingleQuoted()
	case state_AfterDOCTYPE_SystemIdentifier:
		err = t.state_AfterDOCTYPE_SystemIdentifer()

	case state_BogusDOCTYPE:
		err = t.state_Bogus_DOCTYPE()

	case state_CDATA_Section:
		err = t.state_CDATA_Section()
	case state_CDATA_SectionBracket:
		err = t.state_CDATA_SectionBracket()
	case state_CDATA_SectionEnd:
		err = t.state_CDATA_SectionEnd()

	case state_CharacterReference:
		err = t.state_CharacterReference()
	case state_NamedCharacterReference:
		err = t.state_NamedCharacterReference()
	case state_AmbiguousAmpersand:
		err = t.state_AmbiguousAmpersand()

	case state_NumericCharacterReference:
		err = t.state_NumericCharacterReference()
	case state_HexadecimalCharacterReferenceStart:
		err = t.state_HexadecimalCharacterReferenceStart()
	case state_DecimalCharacterReferenceStart:
		err = t.state_DecimalCharacterReferenceStart()
	case state_HexadecimalCharacterReference:
		err = t.state_HexadecimalCharacterReference()
	case state_DecimalCharacterReference:
		err = t.state_DeciamalCharacterReference()
	case state_NumericCharacterReferenceEnd:
		err = t.state_NumericCharacterReferenceEnd()
	}

	return err
}

func (t *Tokenizer) SetState(state int) {
	t.state = state
}

func (t *Tokenizer) ReconsumeToken(token Token) {
	t.tokens = slices.Insert(t.tokens, 0, token)
}

func (t *Tokenizer) ConsumeToken() Token {
	if len(t.tokens) == 0 {
		return nil
	}

	x, a := t.tokens[0], t.tokens[1:]
	t.tokens = a

	return x
}

//#region private

//#region Support

func wasConsumedAsPartOfAttribute(state dgo.Option[int]) bool {
	if state.IsNone() {
		return false
	}
	value := *state.Some

	return value == state_AttributValue_DoubleQuoted ||
		value == state_AttributValue_SingleQuoted ||
		value == state_AttributeName
}

// When a state says to flush code points consumed as a character reference,
// it means that for each code point in the temporary buffer (in the order they were added to the buffer),
// the user agent must append the code point from the buffer to the current attribute's value if the character reference
// was consumed as part of an attribute, or emit the code point as a character token otherwise.
func (t *Tokenizer) flush() {
	if wasConsumedAsPartOfAttribute(t.rstate) {

		tag, ok := t.workingToken.(TagToken)
		if ok {
			tag.AppendStringAttrValue(t.tempbuffer)
		}
		return
	}

	for _, char := range t.tempbuffer {
		t.tokens = append(t.tokens, NewTokenCharacter(char))
	}
}

func (t *Tokenizer) consume() (rune, error) {
	r, _, err := t.reader.ReadRune()
	if err != nil {
		return utf8.RuneError, err
	}

	if r != '\r' {
		return r, nil
	}

	rs, err := t.reader.Peek(1)
	if err != nil {
		return utf8.RuneError, err
	}

	if rs[0] == '\n' {
		m := make([]rune, 1)
		_, err := t.reader.Read(m)
		if err != nil {
			return utf8.MaxRune, err
		}
	}

	return '\n', nil
}

func (t *Tokenizer) emitCurrentWithTokens(tokens ...Token) {

	tag, ok := t.workingToken.(TagToken)
	if ok {
		tag.FinishAttr()
		t.tokens = append(t.tokens, tag)
	}

	t.tokens = append(t.tokens, tokens...)
	t.workingToken = nil
}

// An appropriate end tag token is an end tag token whose tag name matches the tag name of the last start tag to have been
// emitted from this tokenizer, if any. If no start tag has been emitted from this tokenizer,
// then no end tag token is appropriate.
func (t *Tokenizer) hasApproriateEndTagToken() bool {
	tag, ok := t.workingToken.(TokenEndTag)
	if !ok {
		return false
	}

	return t.lastStartTag.IsSome() && *t.lastStartTag.Some == tag.name
}

//#endregion

//#region Entry

// https://html.spec.whatwg.org/#data-state
func (t *Tokenizer) state_Data() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if !isEOF && err != nil {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewTokenEOF())
		return nil
	}

	if char == '&' {
		t.rstate = dgo.Some(t.state)
		t.state = state_CharacterReference
		return nil
	}

	if char == '<' {
		t.state = state_TagOpen
		return nil
	}

	// unexpected-null-character parse error
	if char == '\u0000' {
		t.tokens = append(t.tokens, NewTokenCharacter(char))
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter(char))

	return nil
}

// https://html.spec.whatwg.org/#rcdata-state
func (t *Tokenizer) state_RCData() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if !isEOF && err != nil {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewTokenEOF())
		return nil
	}

	if char == '&' {
		t.rstate = dgo.Some(State_RCData)
		t.state = state_CharacterReference
		return nil
	}

	if char == '<' {
		t.state = state_RCData_LessThanSign
		return nil
	}

	// unexpected-null-character parse error.
	if char == '\u0000' {
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter(char))

	return nil
}

// https://html.spec.whatwg.org/#rawtext-state
func (t *Tokenizer) state_RawText() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if !isEOF && err != nil {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewTokenEOF())
		return nil
	}

	if char == '<' {
		t.state = state_RawText_LessThanSign
		return nil
	}

	//  unexpected-null-character parse error.
	if char == '\u0000' {
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter(char))

	return nil
}

// https://html.spec.whatwg.org/#script-data-state
func (t *Tokenizer) state_ScriptData() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if !isEOF && err != nil {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewTokenEOF())
		return nil
	}

	if char == '<' {
		t.state = state_ScriptData_LessThanSign
		return nil
	}

	// unexpected-null-character parse error.
	if char == '\u0000' {
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter(char))

	return nil
}

// https://html.spec.whatwg.org/#plaintext-state
func (t *Tokenizer) state_PlainText() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if !isEOF && err != nil {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewTokenEOF())
		return nil
	}

	// unexpected-null-character parse error.
	if char == '\u0000' {
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter(char))

	return nil
}

//#endregion Entry

//#region Tag

// https://html.spec.whatwg.org/#tag-open-state
func (t *Tokenizer) state_TagOpen() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if !isEOF && err != nil {
		return err
	}

	// eof-before-tag-name parse error.
	if isEOF {
		t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenEOF())
		return nil
	}

	if char == '!' {
		t.state = state_MarkupDeclarationOpen
		return nil
	}

	if char == '/' {
		t.state = state_EndTagOpen
		return nil
	}

	if unicode.IsLetter(char) {
		t.state = state_TagName
		t.workingToken = NewTokenStartTag("", make(AttributesMap), dgo.None[bool](), "", "")
		return t.reader.UnreadRune()
	}

	//  unexpected-question-mark-instead-of-tag-name parse error.
	if char == '?' {
		t.workingToken = NewTokenComment("")
		t.state = state_BogusComment
		return t.reader.UnreadRune()
	}

	//  invalid-first-character-of-tag-name parse error.

	t.tokens = append(t.tokens, NewTokenCharacter('>'))
	t.state = State_Data
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#end-tag-open-state
func (t *Tokenizer) state_EndTagOpen() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	//  eof-before-tag-name parse error.
	if isEOF {
		t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('/'), NewTokenEOF())
		return nil
	}

	if unicode.IsLetter(char) {
		t.workingToken = NewTokenEndTag("")
		t.state = state_TagName
		return t.reader.UnreadRune()
	}

	// missing-end-tag-name parse error.
	if char == '>' {
		t.state = State_Data
		return nil
	}

	// invalid-first-character-of-tag-name parse error.
	t.workingToken = NewTokenComment("")

	t.state = state_BogusComment
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#tag-name-state
func (t *Tokenizer) state_TagName() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if !isEOF && err != nil {
		return err
	}

	// eof-in-tag parse error.
	if isEOF {
		t.tokens = append(t.tokens, NewTokenEOF())
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		t.state = state_BeforeAttributeName
		return nil
	}

	if char == '/' {
		t.state = state_SelfClosingStartTag
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if unicode.IsLetter(char) && unicode.IsUpper(char) {
		if tag, ok := t.workingToken.(TagToken); ok {
			tag.Name += string(unicode.ToLower(char))
		}
		return nil
	}

	// unexpected-null-character parse error.
	if char == '\u0000' {
		if tag, ok := t.workingToken.(TagToken); ok {
			tag.Name += string(utf8.RuneError)
		}
		return nil
	}

	if tag, ok := t.workingToken.(TagToken); ok {
		tag.Name += string(char)
	}
	return nil
}

//#endregion

//#region RCDATA

// https://html.spec.whatwg.org/#rcdata-less-than-sign-state
func (t *Tokenizer) state_RCData_LessThen() error {
	char, err := t.consume()
	if err != nil {
		return err
	}

	if char == '/' {
		t.tempbuffer = ""
		t.state = state_RCData_EndTagOpen
		return nil
	}

	t.state = State_RCData
	t.tokens = append(t.tokens, NewTokenCharacter('<'))

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#rcdata-end-tag-open-state
func (t *Tokenizer) state_RCData_EndTagOpen() error {
	char, err := t.consume()

	if err != nil {
		return err
	}

	if unicode.IsLetter(char) {
		t.workingToken = NewTokenEndTag("")

		t.state = state_RCData_EndTagName
		return t.reader.UnreadRune()
	}

	t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('/'))
	t.state = State_RCData

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#rcdata-end-tag-name-state
func (t *Tokenizer) state_RCData_EndTagName() error {
	char, err := t.consume()

	if err != nil {
		return err
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		if t.hasApproriateEndTagToken() {
			t.state = state_BeforeAttributeName
			return nil
		}
	}

	if char == '/' {
		if t.hasApproriateEndTagToken() {
			t.state = state_SelfClosingStartTag
			return nil
		}
	}

	if char == '>' {
		if t.hasApproriateEndTagToken() {
			t.state = State_Data
			t.emitCurrentWithTokens()
			return nil
		}
	}

	if unicode.IsLetter(char) {
		if unicode.IsUpper(char) {
			t.tempbuffer += string(char)

			if tag, ok := t.workingToken.(TagToken); ok {
				tag.Name += string(unicode.ToLower(char))
			}

			return nil
		}

		t.tempbuffer += string(char)
		if tag, ok := t.workingToken.(TagToken); ok {
			tag.Name += string(char)
		}
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('/'))

	for _, char := range t.tempbuffer {
		t.tokens = append(t.tokens, NewTokenCharacter(char))
	}

	t.state = State_RCData
	return t.reader.UnreadRune()
}

//#endregion

//#region RAWTEXT

// https://html.spec.whatwg.org/#rawtext-less-than-sign-state
func (t *Tokenizer) state_RawText_LessThenSign() error {
	char, err := t.consume()

	if err != nil {
		return err
	}

	if char == '/' {
		t.tempbuffer = ""
		t.state = state_RawText_EndTagOpen
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter('<'))
	t.state = State_RawText

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#rawtext-end-tag-open-state
func (t *Tokenizer) state_RawText_EndTagOpen() error {
	char, err := t.consume()

	if err != nil {
		return err
	}

	if unicode.IsLetter(char) {
		t.workingToken = NewTokenEndTag("")
		t.state = state_RawText_EndTagName

		return t.reader.UnreadRune()
	}

	t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('/'))
	t.state = State_RawText

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#rawtext-end-tag-name-state
func (t *Tokenizer) state_RawText_EndTagName() error {
	char, err := t.consume()

	if err != nil {
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		if t.hasApproriateEndTagToken() {
			t.state = state_BeforeAttributeName
			return nil
		}
	}

	if char == '/' {
		if t.hasApproriateEndTagToken() {
			t.state = state_SelfClosingStartTag
			return nil
		}
	}

	if char == '>' {
		if t.hasApproriateEndTagToken() {
			t.state = State_Data
			t.emitCurrentWithTokens()
			return nil
		}
	}

	if unicode.IsLetter(char) {
		if unicode.IsUpper(char) {
			t.tempbuffer += string(char)
			if tag, ok := t.workingToken.(TagToken); ok {
				tag.Name += string(unicode.ToLower(char))
			}
			return nil
		}

		t.tempbuffer += string(char)
		if tag, ok := t.workingToken.(TagToken); ok {
			tag.Name += string(char)
		}
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('/'))
	for _, char := range t.tempbuffer {
		t.tokens = append(t.tokens, NewTokenCharacter(char))
	}

	t.state = State_RawText
	return t.reader.UnreadRune()
}

//#endregion

// #region Script
// https://html.spec.whatwg.org/#script-data-less-than-sign-state
func (t *Tokenizer) state_ScriptData_LessThanSign() error {
	char, err := t.consume()

	if err != nil {
		return err
	}

	if char == '/' {
		t.tempbuffer = ""
		t.state = state_ScriptData_EndTagOpen
		return nil
	}

	if char == '!' {
		t.state = state_ScriptData_EscapeStart
		t.tokens = append(t.tokens, NewCharacterToken('<'), NewCharacterToken('!'))
		return nil
	}

	t.tokens = append(t.tokens, NewCharacterToken('<'))
	t.state = State_ScriptData

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-end-tag-open-state
func (t *Tokenizer) state_ScriptData_EndTagOpen() error {
	char, err := t.consume()
	if err != nil {
		return nil
	}

	if unicode.IsLetter(char) {
		tag := NewEndToken()
		tag.Name = ""
		t.workingToken = tag
		t.state = state_ScriptData_EndTagName
		return t.reader.UnreadRune()
	}

	t.tokens = append(t.tokens, NewCharacterToken('<'), NewCharacterToken('/'))
	t.state = State_ScriptData

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-end-tag-name-state
func (t *Tokenizer) state_ScriptData_EndTagName() error {
	char, err := t.consume()
	if err != nil {
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		if t.hasApproriateEndTagToken() {
			t.state = state_BeforeAttributeName
			return nil
		}
	}

	if char == '/' {
		if t.hasApproriateEndTagToken() {
			t.state = state_SelfClosingStartTag
			return nil
		}
	}

	if char == '>' {
		if t.hasApproriateEndTagToken() {
			t.state = State_Data
			t.emitCurrentWithTokens()
			return nil
		}
	}

	if unicode.IsLetter(char) {
		if unicode.IsUpper(char) {
			t.tempbuffer += string(char)
			if tag, ok := t.workingToken.(TagToken); ok {
				tag.Name += string(unicode.ToLower(char))
			}
			return nil
		}

		t.tempbuffer += string(char)
		if tag, ok := t.workingToken.(TagToken); ok {
			tag.Name += string(char)
		}
		return nil
	}

	t.tokens = append(t.tokens, NewCharacterToken('<'), NewCharacterToken('/'))
	for _, char := range t.tempbuffer {
		t.tokens = append(t.tokens, NewCharacterToken(char))
	}

	t.state = State_ScriptData
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-escape-start-state
func (t *Tokenizer) state_ScriptData_Escape_Start() error {
	char, err := t.consume()
	if err != nil {
		return nil
	}

	if char == '-' {
		t.state = state_ScriptData_EscapeStartDash
		t.tokens = append(t.tokens, NewCharacterToken('-'))
		return nil
	}

	t.state = State_ScriptData
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-escape-start-dash-state
func (t *Tokenizer) state_ScriptData_Escape_StartDash() error {
	char, err := t.consume()
	if err != nil {
		return nil
	}

	if char == '-' {
		t.state = state_ScriptData_EscapedDashDash
		t.tokens = append(t.tokens, NewCharacterToken('-'))
		return nil
	}

	t.state = State_ScriptData
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-escaped-state
func (t *Tokenizer) state_ScriptData_Escaped() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return nil
	}

	if char == '-' {
		t.state = state_ScriptData_EscapedDash
		t.tokens = append(t.tokens, NewCharacterToken('-'))
		return nil
	}

	if char == '<' {
		t.state = state_ScriptData_EscapedLessThanSign
		return nil
	}

	if char == '\u0000' {
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewCharacterToken(char))

	return nil
}

// https://html.spec.whatwg.org/#script-data-escaped-dash-state
func (t *Tokenizer) state_ScriptData_Escaped_Dash() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return nil
	}

	if char == '-' {
		t.state = state_ScriptData_EscapedDashDash
		t.tokens = append(t.tokens, NewCharacterToken('-'))
		return nil
	}

	if char == '<' {
		t.state = state_ScriptData_EscapedLessThanSign
		return nil
	}

	if char == '\u0000' {
		t.state = state_ScriptData_Escaped
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.state = state_ScriptData_Escaped
	t.tokens = append(t.tokens, NewCharacterToken(char))

	return nil
}

// https://html.spec.whatwg.org/#script-data-escaped-dash-dash-state
func (t *Tokenizer) state_ScriptData_Escaped_DashDash() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return nil
	}

	if char == '-' {
		t.tokens = append(t.tokens, NewCharacterToken('-'))
		return nil
	}

	if char == '<' {
		t.state = state_ScriptData_EscapedLessThanSign
		return nil
	}

	if char == '>' {
		t.state = State_ScriptData
		t.tokens = append(t.tokens, NewCharacterToken('>'))
		return nil
	}

	if char == '\u0000' {
		t.state = state_ScriptData_Escaped
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.state = state_ScriptData_Escaped
	t.tokens = append(t.tokens, NewCharacterToken(char))

	return nil
}

// https://html.spec.whatwg.org/#script-data-escaped-less-than-sign-state
func (t *Tokenizer) state_ScriptData_Escaped_LessThanSign() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if char == '/' {
		t.tempbuffer = ""
		t.state = state_ScriptData_EscapedEndTagOpen
		return nil
	}

	if unicode.IsLetter(char) {
		t.tempbuffer = ""
		t.tokens = append(t.tokens, NewCharacterToken('<'))
		t.state = state_ScriptData_DoubleEscapedStart
		return t.reader.UnreadRune()
	}

	t.tokens = append(t.tokens, NewCharacterToken('<'))
	t.state = state_ScriptData_Escaped
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-escaped-end-tag-open-state
func (t *Tokenizer) state_ScriptData_Escaped_EndTagOpen() error {
	char, err := t.consume()
	if err != nil {
		return nil
	}

	if unicode.IsLetter(char) {
		t.emitCurrentWithTokens()
		t.state = state_ScriptData_EscapedEndTagName

		return t.reader.UnreadRune()
	}

	t.tokens = append(t.tokens, NewCharacterToken('<'), NewCharacterToken('/'))
	t.state = state_ScriptData_Escaped

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-escaped-end-tag-name-state
func (t *Tokenizer) state_ScriptData_Escaped_EndTagName() error {
	char, err := t.consume()
	if err != nil {
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		if t.hasApproriateEndTagToken() {
			t.state = state_BeforeAttributeName
			return nil
		}
	}

	if char == '/' {
		if t.hasApproriateEndTagToken() {
			t.state = state_SelfClosingStartTag
			return nil
		}
	}

	if char == '>' {
		if t.hasApproriateEndTagToken() {
			t.state = State_Data
			t.emitCurrentWithTokens()
			return nil
		}
	}

	if unicode.IsLetter(char) {
		if unicode.IsUpper(char) {
			t.tempbuffer += string(char)
			if tag, ok := t.workingToken.(TagToken); ok {
				tag.Name += string(unicode.ToLower(char))
			}
			return nil
		}

		t.tempbuffer += string(char)
		if tag, ok := t.workingToken.(TagToken); ok {
			tag.Name += string(char)
		}
		return nil
	}

	t.tokens = append(t.tokens, NewCharacterToken('<'), NewCharacterToken('/'))

	for _, char := range t.tempbuffer {
		t.tokens = append(t.tokens, NewCharacterToken(char))
	}

	t.state = state_ScriptData_Escaped

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-double-escape-start-state
func (t *Tokenizer) state_ScriptData_DoubleEscapedStart() error {
	char, err := t.consume()
	if err != nil {
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' ||
		char == ' ' || char == '/' || char == '>' {

		if t.tempbuffer == "script" {
			t.state = state_ScriptData_DoubleEscaped
			return nil
		}

		t.state = state_ScriptData_Escaped
		t.tokens = append(t.tokens, NewCharacterToken(char))
		return nil
	}

	if unicode.IsLetter(char) {
		if unicode.IsUpper(char) {
			t.tempbuffer += string(unicode.ToLower(char))
		} else {
			t.tempbuffer += string(char)
		}

		t.tokens = append(t.tokens, NewCharacterToken(char))

		return nil
	}

	t.state = state_ScriptData_Escaped

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-double-escaped-state
func (t *Tokenizer) state_ScriptData_DoubleEscaped() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return nil
	}

	if char == '-' {
		t.state = state_ScriptData_DoubleEscapedDash
		t.tokens = append(t.tokens, NewCharacterToken('-'))
		return nil
	}

	if char == '<' {
		t.state = state_ScriptData_DoubleEscapedLessThanSign
		t.tokens = append(t.tokens, NewCharacterToken('<'))
		return nil
	}

	if char == '\u0000' {
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewCharacterToken(char))

	return nil
}

// https://html.spec.whatwg.org/#script-data-double-escaped-dash-state
func (t *Tokenizer) state_ScriptData_DoubleEscapedDash() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return nil
	}

	if char == '-' {
		t.state = state_ScriptData_DoubleEscapedDashDash
		t.tokens = append(t.tokens, NewCharacterToken('-'))
		return nil
	}

	if char == '<' {
		t.state = state_ScriptData_DoubleEscapedLessThanSign
		t.tokens = append(t.tokens, NewCharacterToken('<'))
		return nil
	}

	if char == '\u0000' {
		t.state = state_ScriptData_Escaped
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.state = state_ScriptData_DoubleEscaped
	t.tokens = append(t.tokens, NewCharacterToken(char))

	return nil
}

// https://html.spec.whatwg.org/#script-data-double-escaped-dash-dash-state
func (t *Tokenizer) state_ScriptData_DoubleEscapedDashDash() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return nil
	}

	if char == '-' {
		t.tokens = append(t.tokens, NewCharacterToken('-'))
		return nil
	}

	if char == '<' {
		t.tokens = append(t.tokens, NewCharacterToken('<'))
		t.state = state_ScriptData_DoubleEscapedLessThanSign
		return nil
	}

	if char == '>' {
		t.state = State_ScriptData
		t.tokens = append(t.tokens, NewCharacterToken('>'))
		return nil
	}

	if char == '\u0000' {
		t.state = state_ScriptData_DoubleEscaped
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.state = state_ScriptData_DoubleEscaped
	t.tokens = append(t.tokens, NewCharacterToken(char))

	return nil
}

// https://html.spec.whatwg.org/#script-data-double-escaped-less-than-sign-state
func (t *Tokenizer) state_ScriptData_DoubleEscaped_LessThanSign() error {
	char, err := t.consume()
	if err != nil {
		return nil
	}

	if char == '/' {
		t.tempbuffer = ""
		t.state = state_ScriptData_DoubleEscapeEnd
		t.tokens = append(t.tokens, NewCharacterToken('/'))
		return nil
	}

	t.state = state_ScriptData_DoubleEscaped

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-double-escape-end-state
func (t *Tokenizer) state_ScriptData_DoubleEscapedEnd() error {
	char, err := t.consume()
	if err != nil {
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' ||
		char == ' ' || char == '/' || char == '>' {
		if t.tempbuffer == "script" {
			t.state = state_ScriptData_Escaped
			return nil
		}

		t.state = state_ScriptData_DoubleEscaped
		t.tokens = append(t.tokens, NewCharacterToken(char))
		return nil
	}

	if unicode.IsLetter(char) {
		if unicode.IsUpper(char) {
			t.tempbuffer += string(unicode.ToLower(char))
		} else {
			t.tempbuffer += string(char)
		}

		t.tokens = append(t.tokens, NewCharacterToken(char))
		return nil
	}

	t.state = state_ScriptData_DoubleEscaped

	return t.reader.UnreadRune()
}

//#endregion
// #region Attr

// https://html.spec.whatwg.org/#before-attribute-name-state
func (t *Tokenizer) state_BeforeAttributeName() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF || char == '/' || char == '>' {
		t.state = state_AfterAttributeName
		return t.reader.UnreadRune()
	}

	if char == '\t' || char == '\n' ||
		char == '\f' || char == ' ' {
		return nil
	}

	// unexpected-equals-sign-before-attribute-name parse error
	if char == '=' {
		t.state = state_AttributeName

		if tag, ok := t.workingToken.(TagToken); ok {
			tag.NewAttr(string(char), "")
		}
		return nil
	}

	if tag, ok := t.workingToken.(TagToken); ok {
		tag.NewAttr("", "")
	}

	t.state = state_AttributeName

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#attribute-name-state
func (t *Tokenizer) state_AttributeName() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF || char == '\t' || char == '\n' || char == '\f' ||
		char == ' ' || char == '/' || char == '>' {
		t.state = state_AfterAttributeName
		return t.reader.UnreadRune()
	}

	if char == '=' {
		t.state = state_BeforeAttributeValue
		return nil
	}

	if unicode.IsLetter(char) && unicode.IsUpper(char) {
		if tag, ok := t.workingToken.(TagToken); ok {
			tag.AppendStringToAttrName(string(unicode.ToLower(char)))
		}
		return nil
	}

	// unexpected-null-character parse error.
	if char == '\u0000' {
		if tag, ok := t.workingToken.(TagToken); ok {
			tag.AppendStringToAttrName(string(utf8.RuneError))
		}
		return nil
	}

	// This is an unexpected-character-in-attribute-name parse error. Treat it as per the "anything else" entry below.
	//if char == '"' || char == '\'' || char == '<' {}

	if tag, ok := t.workingToken.(TagToken); ok {
		tag.AppendStringToAttrName(string(char))
	}

	return nil
}

// https://html.spec.whatwg.org/#after-attribute-name-state
func (t *Tokenizer) state_AfterAttributeName() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	//  eof-in-tag parse error.
	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		return nil
	}

	if char == '/' {
		t.state = state_SelfClosingStartTag
		return nil
	}

	if char == '=' {
		t.state = state_BeforeAttributeValue
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if tag, ok := t.workingToken.(TagToken); ok {
		tag.NewAttr("", "")
	}

	t.state = state_AttributeName
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#before-attribute-value-state
func (t *Tokenizer) state_BeforeAttributeValue() error {
	char, err := t.consume()
	if err != nil {
		return err
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		return nil
	}

	if char == '"' {
		t.state = state_AttributValue_DoubleQuoted
		return nil
	}

	if char == '\'' {
		t.state = state_AttributValue_SingleQuoted
		return nil
	}

	// missing-attribute-value parse error
	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	t.state = state_AttributValue_Unquoted

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#attribute-value-(double-quoted)-state
func (t *Tokenizer) state_AttributeValue_DoubleQuote() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return nil
	}

	if char == '"' {
		t.state = state_AfterAttributeValue_Quoted
		return nil
	}

	if char == '&' {
		t.rstate = state_AttributValue_DoubleQuoted
		t.state = state_CharacterReference
		return nil
	}

	if char == '\u0000' {
		if tag, ok := t.workingToken.(TagToken); ok {
			tag.AppendStringAttrValue(string(utf8.RuneError))
		}
		return nil
	}

	if tag, ok := t.workingToken.(TagToken); ok {
		tag.AppendStringAttrValue(string(char))
	}
	return nil
}

// https://html.spec.whatwg.org/#attribute-value-(single-quoted)-state
func (t *Tokenizer) state_AttributeValue_SignleQuote() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return nil
	}

	if char == '\'' {
		t.state = state_AfterAttributeValue_Quoted
		return nil
	}

	if char == '&' {
		t.rstate = state_AttributValue_SingleQuoted
		t.state = state_CharacterReference
		return nil
	}

	if char == '\u0000' {
		if tag, ok := t.workingToken.(TagToken); ok {
			tag.AppendStringAttrValue(string(utf8.RuneError))
		}
		return nil
	}

	if tag, ok := t.workingToken.(TagToken); ok {
		tag.AppendStringAttrValue(string(char))
	}
	return nil
}

// https://html.spec.whatwg.org/#attribute-value-(unquoted)-state
func (t *Tokenizer) state_AttributeValue_Unquoted() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		t.state = state_BeforeAttributeName
		return nil
	}

	if char == '&' {
		t.rstate = state_AttributValue_Unquoted
		t.state = state_CharacterReference
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if char == '\u0000' {
		if tag, ok := t.workingToken.(TagToken); ok {
			tag.AppendStringAttrValue(string(utf8.RuneError))
		}
		return nil
	}

	//  unexpected-character-in-unquoted-attribute-value parse error. Treat it as per the "anything else" entry below.
	//if char == '"' || char == '\'' || char == '<' || char == '=' || char == '`' {}
	if tag, ok := t.workingToken.(TagToken); ok {
		tag.AppendStringAttrValue(string(char))
	}
	return nil
}

// https://html.spec.whatwg.org/#after-attribute-value-(quoted)-state
func (t *Tokenizer) state_AfterAttributeValue_Quoted() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		t.state = state_BeforeAttributeName
		return nil
	}

	if char == '/' {
		t.state = state_SelfClosingStartTag
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	//  missing-whitespace-between-attributes parse error.
	t.state = state_BeforeAttributeName
	return t.reader.UnreadRune()
}

//#endregion
//#endregion

// https://html.spec.whatwg.org/#self-closing-start-tag-state
func (t *Tokenizer) state_SelfClosingStartTag() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return nil
	}

	if char == '>' {
		if tag, ok := t.workingToken.(TagToken); ok {
			*tag.Selfclosing = true
		}
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}
	t.state = state_BeforeAttributeName
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#bogus-comment-state
func (t *Tokenizer) state_BogusComment() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.emitCurrentWithTokens(NewEOFToken())
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if char == '\u0000' {
		if tag, ok := t.workingToken.(*CommentToken); ok {
			tag.Data += string(utf8.RuneError)
		}
		return nil
	}

	if tag, ok := t.workingToken.(*CommentToken); ok {
		tag.Data += string(char)
	}
	return nil
}

// https://html.spec.whatwg.org/#markup-declaration-open-state
func (t *Tokenizer) state_MarkupDeclarationOpen() error {

	runes, err := t.reader.Peek(2)
	if err != nil {
		return err
	}

	if runes[0] == '-' && runes[1] == '-' {
		out := make([]rune, 2)
		_, err := t.reader.Read(out)
		if err != nil {
			return err
		}

		t.state = state_CommentStart
		t.workingToken = NewCommentToken()

		return nil
	}

	runes, err = t.reader.Peek(7)
	if err != nil {
		return err
	}

	a := string(runes)
	if strings.ToUpper(a) == "DOCTYPE" {
		t.state = state_DOCTYPE

		rs := make([]rune, 7)
		_, err := t.reader.Read(rs)

		if err != nil {
			return nil
		}

		return nil
	}

	if a == "[CDATA[" {
		rs := make([]rune, 7)
		_, err := t.reader.Read(rs)
		if err != nil {
			return err
		}

		//TODO: adjeusted current node check && html namespace check
		// t.state = State_CDATA_Section

		tag := NewCommentToken()
		tag.Data += a
		t.workingToken = tag
		t.state = state_BogusComment
		return nil
	}

	t.workingToken = NewCommentToken()
	t.state = state_BogusComment

	return nil
}

//#endregion
//#region Comment

// https://html.spec.whatwg.org/#comment-start-state
func (t *Tokenizer) state_CommentStart() error {
	char, err := t.consume()

	if err != nil {
		return err
	}

	if char == '-' {
		t.state = state_CommentStartDash
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	t.state = state_Comment

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#comment-start-dash-state
func (t *Tokenizer) state_Comment_StartDash() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.emitCurrentWithTokens(NewEOFToken())
		return nil
	}

	if char == '-' {
		t.state = state_CommentEnd
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if tag, ok := t.workingToken.(*CommentToken); ok {
		tag.Data += "-"
	}
	t.state = state_Comment

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#comment-state
func (t *Tokenizer) state_Comment() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.emitCurrentWithTokens(NewEOFToken())
		return nil
	}

	if char == '<' {
		if tag, ok := t.workingToken.(*CommentToken); ok {
			tag.Data += string(char)
		}
		t.state = state_CommentLessThanSign
		return nil
	}

	if char == '-' {
		t.state = state_CommentEndDash
		return nil
	}

	if char == '\u0000' {
		if tag, ok := t.workingToken.(*CommentToken); ok {
			tag.Data += string(utf8.RuneError)
		}
		return nil
	}

	if tag, ok := t.workingToken.(*CommentToken); ok {
		tag.Data += string(char)
	}

	return nil
}

// https://html.spec.whatwg.org/#comment-less-than-sign-state
func (t *Tokenizer) state_Comment_LessThanSign() error {
	char, err := t.consume()

	if err != nil {
		return err
	}

	if char == '!' {
		if tag, ok := t.workingToken.(*CommentToken); ok {
			tag.Data += string(char)
		}
		t.state = state_CommentLessThanSignBang
		return nil
	}

	if char == '<' {
		if tag, ok := t.workingToken.(*CommentToken); ok {
			tag.Data += string(char)
		}
		return nil
	}

	t.state = state_Comment
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#comment-less-than-sign-bang-state
func (t *Tokenizer) state_Comment_LessThanSign_Bang() error {
	char, err := t.consume()

	if err != nil {
		return err
	}

	if char == '-' {
		t.state = state_CommentLessThanSignBangDash
		return nil
	}

	t.state = state_Comment

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#comment-less-than-sign-bang-dash-state
func (t *Tokenizer) state_Comment_LessThanSign_BangDash() error {
	char, err := t.consume()

	if err != nil {
		return err
	}

	if char == '-' {
		t.state = state_CommentLessThanSignBangDashDash
		return nil
	}

	t.state = state_CommentEndDash
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#comment-less-than-sign-bang-dash-dash-state
func (t *Tokenizer) state_Comment_LessThanSign_BangDashDash() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF || char == '>' {
		t.state = state_CommentEnd
		return t.reader.UnreadRune()
	}

	// This is a nested-comment parse error.
	t.state = state_CommentEnd

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#comment-end-dash-state
func (t *Tokenizer) state_Comment_EndDash() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.emitCurrentWithTokens(NewEOFToken())
		return nil
	}

	if char == '-' {
		t.state = state_CommentEnd
		return nil
	}
	if tag, ok := t.workingToken.(*CommentToken); ok {
		tag.Data += "-"
	}
	t.state = state_Comment

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#comment-end-state
func (t *Tokenizer) state_CommentEnd() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.emitCurrentWithTokens(NewEOFToken())
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if char == '!' {
		t.state = state_CommentEndBang
		return nil
	}

	if char == '-' {
		if tag, ok := t.workingToken.(*CommentToken); ok {
			tag.Data += string(char)
		}
		return nil
	}

	if tag, ok := t.workingToken.(*CommentToken); ok {
		tag.Data += "--"
	}
	t.state = state_Comment

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#comment-end-bang-state
func (t *Tokenizer) state_Comment_EndBang() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.emitCurrentWithTokens(NewEOFToken())
		return nil
	}

	if char == '-' {
		if tag, ok := t.workingToken.(*CommentToken); ok {
			tag.Data += "--!"
		}
		t.state = state_CommentEndDash
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if tag, ok := t.workingToken.(*CommentToken); ok {
		tag.Data += "--!"
	}
	t.state = state_Comment

	return t.reader.UnreadRune()
}

//#endregion
// #region DOCTYPE

// https://html.spec.whatwg.org/#doctype-state
func (t *Tokenizer) state_DOCTYPE() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		tag := NewDOCTYPEToken()
		tag.ForceQuirks = true
		t.workingToken = tag
		t.emitCurrentWithTokens(NewEOFToken())
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		t.state = state_BeforeDOCTYPEName
		return nil
	}

	if char == '>' {
		t.state = state_BeforeDOCTYPEName
		return t.reader.UnreadRune()
	}

	t.state = state_BeforeDOCTYPEName
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#before-doctype-name-state
func (t *Tokenizer) state_BeforeDOCTYPEName() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		tag := NewDOCTYPEToken()
		tag.ForceQuirks = true
		t.workingToken = tag
		t.emitCurrentWithTokens(NewEOFToken())
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		return nil
	}

	if unicode.IsLetter(char) && unicode.IsUpper(char) {
		tag := NewDOCTYPEToken()
		*tag.Name = string(unicode.ToLower(char))
		t.state = state_DOCTYPE_Name
		return nil
	}

	if char == '\u0000' {
		tag := NewDOCTYPEToken()
		*tag.Name = string(utf8.RuneError)
		t.state = state_DOCTYPE_Name
		return nil
	}

	if char == '>' {
		tag := NewDOCTYPEToken()
		tag.ForceQuirks = true
		t.workingToken = tag
		t.state = State_Data
		return nil
	}

	tag := NewDOCTYPEToken()
	*tag.Name = string(char)
	t.workingToken = tag

	t.state = state_DOCTYPE_Name
	return nil
}

// https://html.spec.whatwg.org/#doctype-name-state
func (t *Tokenizer) state_DOCTYPE_Name() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			tag.ForceQuirks = true
		}

		t.emitCurrentWithTokens(NewEOFToken())
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		t.state = state_AfterDOCTYPE_Name
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if unicode.IsLetter(char) && unicode.IsUpper(char) {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			*tag.Name += string(unicode.ToLower(char))
		}
		return nil
	}

	if char == '\u0000' {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			*tag.Name += string(utf8.RuneError)
		}
		return nil
	}
	if tag, ok := t.workingToken.(*DoctypeToken); ok {
		*tag.Name += string(char)
	}

	return nil
}

// https://html.spec.whatwg.org/#after-doctype-name-state
func (t *Tokenizer) state_AfterDOCTYPE_Name() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			tag.ForceQuirks = true
		}
		t.emitCurrentWithTokens(NewEOFToken())
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	ptest, err := t.reader.Peek(5)
	if err != nil {
		return err
	}

	text := strings.ToUpper(string(char) + string(ptest))

	if text == "PUBLIC" {
		v := make([]rune, 5)

		_, err = t.reader.Read(v)
		if err != nil {
			return err
		}

		t.state = state_AfterDOCTYPE_PublicKeyword
		return nil
	}

	if text == "SYSTEM" {
		v := make([]rune, 5)

		_, err = t.reader.Read(v)
		if err != nil {
			return err
		}

		t.state = state_AfterDOCTYPE_SystemKeyword
		return nil
	}

	t.state = state_BogusDOCTYPE
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#after-doctype-public-keyword-state
func (t *Tokenizer) state_AfterDOCTYPE_PublicKeyword() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			tag.ForceQuirks = true
		}
		t.emitCurrentWithTokens(NewEOFToken())
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		t.state = state_BeforeDOCTYPE_PublicIdentifier
		return nil
	}

	if char == '"' {
		t.state = state_DOCTYPE_PublicIdentifier_DoubleQuoted
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			*tag.PublicIdentifier = ""
		}
		return nil
	}

	if char == '\'' {
		t.state = state_DOCTYPE_PublicIdentifier_SingleQuoted
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			*tag.PublicIdentifier = ""
		}
		return nil
	}

	if char == '>' {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			tag.ForceQuirks = true
		}
		t.state = State_Data

		t.emitCurrentWithTokens()
		return nil
	}

	if tag, ok := t.workingToken.(*DoctypeToken); ok {
		tag.ForceQuirks = true
	}
	t.state = state_BogusDOCTYPE
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#before-doctype-public-identifier-state
func (t *Tokenizer) state_BeforeDOCTYPE_PublicIdentifier() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			tag.ForceQuirks = true
		}
		t.emitCurrentWithTokens(NewEOFToken())
		t.state = State_Data
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		return nil
	}

	if char == '"' {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			*tag.PublicIdentifier = ""
		}
		t.state = state_DOCTYPE_PublicIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			*tag.PublicIdentifier = ""
		}
		t.state = state_DOCTYPE_PublicIdentifier_SingleQuoted
		return nil
	}

	if char == '>' {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			tag.ForceQuirks = true
		}
		t.emitCurrentWithTokens()
		return nil
	}

	if tag, ok := t.workingToken.(*DoctypeToken); ok {
		tag.ForceQuirks = true
	}
	t.state = state_BogusDOCTYPE
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#doctype-public-identifier-(double-quoted)-state
func (t *Tokenizer) state_DOCTYPE_PublicIdentifier_DoubleQuoted() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			tag.ForceQuirks = true
		}
		t.emitCurrentWithTokens(NewEOFToken())
		return nil
	}

	if char == '"' {
		t.state = state_AfterDOCTYPE_PublicIdentifier
		return nil
	}

	if char == '\u0000' {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			*tag.PublicIdentifier += string(utf8.RuneError)
		}
		return nil
	}

	if char == '>' {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			tag.ForceQuirks = true
		}
		t.emitCurrentWithTokens()
		t.state = State_Data
		return nil
	}
	if tag, ok := t.workingToken.(*DoctypeToken); ok {
		*tag.PublicIdentifier += string(char)
	}

	return nil
}

// https://html.spec.whatwg.org/#doctype-public-identifier-(single-quoted)-state
func (t *Tokenizer) state_DOCKTYPE_PublicIdentifier_SingleQuoted() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			tag.ForceQuirks = true
		}
		t.emitCurrentWithTokens(NewEOFToken())
		return nil
	}

	if char == '\'' {
		t.state = state_AfterDOCTYPE_PublicIdentifier
		return nil
	}

	if char == '\u0000' {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			*tag.PublicIdentifier += string(char)
		}
		return nil
	}

	if char == '>' {
		if tag, ok := t.workingToken.(*DoctypeToken); ok {
			tag.ForceQuirks = true
		}
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}
	if tag, ok := t.workingToken.(*DoctypeToken); ok {
		*tag.PublicIdentifier += string(char)
	}

	return nil
}

// https://html.spec.whatwg.org/#after-doctype-public-identifier-state
func (t *Tokenizer) state_AfterDOCTYPE_PublicIdentifier() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.emitCurrentWithTokens(NewTokenEOF())
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		t.state = state_BetweenDOCTYPE_PublicAndSystemIdentifiers
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if char == '"' {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.systemIdentifier = dgo.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.systemIdentifier = dgo.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_SingleQuoted
		return nil
	}

	if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
		tag.forceQuirks = true
	}
	t.state = state_BogusDOCTYPE

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#between-doctype-public-and-system-identifiers-state
func (t *Tokenizer) state_BetweenDOCTYPE_PublicAndSystemIdent() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.emitCurrentWithTokens(NewTokenEOF())
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if char == '"' {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.systemIdentifier = dgo.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.systemIdentifier = dgo.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_SingleQuoted
		return nil
	}

	if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
		tag.forceQuirks = true
	}
	t.state = state_BogusDOCTYPE

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#after-doctype-system-keyword-state
func (t *Tokenizer) state_AfterDOCTYPE_SystemKeyword() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if !isEOF && err != nil {
		return err
	}

	if isEOF {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.emitCurrentWithTokens(NewTokenEOF())
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		return nil
	}

	if char == '"' {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.systemIdentifier = dgo.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.systemIdentifier = dgo.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_SingleQuoted
		return nil
	}

	if char == '>' {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.state = State_Data

		t.emitCurrentWithTokens()
		return nil
	}

	if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
		tag.forceQuirks = true
	}
	t.state = state_BogusDOCTYPE

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#before-doctype-system-identifier-state
func (t *Tokenizer) state_BeforeDOCTYPE_SystemIdentifier() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.emitCurrentWithTokens(NewTokenEOF())
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		return nil
	}

	if char == '"' {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.systemIdentifier = dgo.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.systemIdentifier = dgo.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_SingleQuoted
		return nil
	}

	if char == '>' {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
		tag.forceQuirks = true
	}
	t.state = state_BogusDOCTYPE

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#doctype-system-identifier-(double-quoted)-state
func (t *Tokenizer) state_DOCTYPE_SystemIdentifier_DoubleQuoted() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.emitCurrentWithTokens(NewTokenEOF())
		return nil
	}

	if char == '"' {
		t.state = state_AfterDOCTYPE_SystemIdentifier
		return nil
	}

	if char == '\u0000' {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			if tag.systemIdentifier.IsSome() {
				*tag.systemIdentifier.Some += string(utf8.RuneError)
			}

		}
		return nil
	}

	if char == '>' {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
		if tag.systemIdentifier.IsSome() {
			*tag.systemIdentifier.Some += string(char)
		}
	}

	return nil
}

// https://html.spec.whatwg.org/#doctype-system-identifier-(single-quoted)-state
func (t *Tokenizer) state_DOCTYPE_SystemIdentifier_SingleQuoted() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.emitCurrentWithTokens(NewTokenEOF())
		return nil
	}

	if char == '\'' {
		t.state = state_AfterDOCTYPE_SystemIdentifier
		return nil
	}

	if char == '\u0000' {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			if tag.systemIdentifier.IsSome() {
				*tag.systemIdentifier.Some += string(utf8.RuneError)
			}
		}
		return nil
	}

	if char == '>' {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
		if tag.systemIdentifier.IsSome() {
			*tag.systemIdentifier.Some += string(char)
		}
	}

	return nil
}

// https://html.spec.whatwg.org/#after-doctype-system-identifier-state
func (t *Tokenizer) state_AfterDOCTYPE_SystemIdentifer() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		if tag, ok := t.workingToken.(TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.emitCurrentWithTokens(NewTokenEOF())
		return nil
	}

	if char == '\t' || char == '\n' || char == '\f' || char == ' ' {
		t.state = state_BeforeDOCTYPE_SystemIdentifer
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	t.state = state_BogusDOCTYPE
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#bogus-doctype-state
func (t *Tokenizer) state_Bogus_DOCTYPE() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.emitCurrentWithTokens(NewTokenEOF())
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	return nil
}

//#endregion
//#region CDATA

// https://html.spec.whatwg.org/#cdata-section-state
func (t *Tokenizer) state_CDATA_Section() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if !isEOF && err != nil {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewTokenEOF())
		return nil
	}

	if char == ']' {
		t.state = state_CDATA_SectionBracket
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter(char))

	return nil
}

// https://html.spec.whatwg.org/#cdata-section-bracket-state
func (t *Tokenizer) state_CDATA_SectionBracket() error {
	char, err := t.consume()
	if err != nil {
		return err
	}

	if char == ']' {
		t.state = state_CDATA_SectionEnd
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter(']'))
	t.state = state_CDATA_Section

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#cdata-section-end-state
func (t *Tokenizer) state_CDATA_SectionEnd() error {
	char, err := t.consume()
	if err != nil {
		return err
	}

	if char == ']' {
		t.tokens = append(t.tokens, NewTokenCharacter(']'))
		return nil
	}

	if char == '>' {
		t.state = State_Data
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter(']'), NewTokenCharacter(']'))
	t.state = state_CDATA_Section

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#character-reference-state
func (t *Tokenizer) state_CharacterReference() error {
	t.tempbuffer = "&"

	char, err := t.consume()
	if err != nil {
		return err
	}

	if unicode.IsNumber(char) || unicode.IsLetter(char) {
		t.state = state_NamedCharacterReference
		return t.reader.UnreadRune()
	}

	if char == '#' {
		t.tempbuffer += "#"
		t.state = state_NumericCharacterReference
		return nil
	}

	t.flush()
	t.state = t.rstate.MustSome()
	return t.reader.UnreadRune()
}

//#endregion
//#region reference

// https://html.spec.whatwg.org/#named-character-reference-state
func (t *Tokenizer) state_NamedCharacterReference() error {

	/*runes, err := t.reader.Peek(25)
	if err != nil {
		return err
	}

	if runes[0] == 'a' && runes[1] == 'c' {
		if wasconsumedAsPartOfAttribute(t.state) && runes[2] != ';' && (runes[2] == '=' || unicode.IsLetter(runes[2]) || unicode.IsNumber(runes[2])) {
			t.flush()
			t.state = t.rstate
			return nil
		}

		if runes[2] != ';' {

		}

		t.tempbuffer = string(rune('∾'))
		t.flush()
		t.state = t.rstate
		return nil
	}*/

	t.state = state_AmbiguousAmpersand
	t.flush()
	return nil
}

// https://html.spec.whatwg.org/#ambiguous-ampersand-state
func (t *Tokenizer) state_AmbiguousAmpersand() error {
	char, err := t.consume()
	if err != nil {
		return err
	}

	if unicode.IsLetter(char) || unicode.IsNumber(char) {
		if wasConsumedAsPartOfAttribute(t.rstate) {

			if tag, ok := t.workingToken.(*TagToken); ok {
				tag.AppendStringAttrValue(string(char))
			}

			return nil
		}

		t.tokens = append(t.tokens, NewTokenCharacter(char))
		return nil
	}

	//  unknown-named-character-reference parse error.
	/*if char == ';' {
		t.state = t.rstate
		return t.reader.UnreadRune()
	}*/

	t.state = t.rstate.MustSome()
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#numeric-character-reference-state
func (t *Tokenizer) state_NumericCharacterReference() error {
	char, err := t.consume()
	if err != nil {
		return err
	}

	t.characterReferenceCode = 0

	if char == 'x' || char == 'X' {
		t.tempbuffer += string(char)
		t.state = state_HexadecimalCharacterReferenceStart
		return nil
	}

	t.state = state_DecimalCharacterReferenceStart

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#hexadecimal-character-reference-start-state
func (t *Tokenizer) state_HexadecimalCharacterReferenceStart() error {
	char, err := t.consume()
	if err != nil {
		return err
	}

	if unicode.Is(unicode.ASCII_Hex_Digit, char) {
		t.state = state_HexadecimalCharacterReference
		return t.reader.UnreadRune()
	}

	t.flush()
	t.state = t.rstate.MustSome()
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#hexadecimal-character-reference-start-state
func (t *Tokenizer) state_DecimalCharacterReferenceStart() error {
	char, err := t.consume()
	if err != nil {
		return err
	}

	if unicode.IsDigit(char) {
		t.state = state_DecimalCharacterReference
		return t.reader.UnreadRune()
	}

	t.flush()
	t.state = t.rstate.MustSome()
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#hexadecimal-character-reference-state
func (t *Tokenizer) state_HexadecimalCharacterReference() error {
	char, err := t.consume()
	if err != nil {
		return err
	}

	if unicode.IsDigit(char) {
		t.characterReferenceCode *= 16
		t.characterReferenceCode += int(char - '0')
		return nil
	}

	if unicode.Is(unicode.ASCII_Hex_Digit, char) {
		if unicode.IsUpper(char) {
			t.characterReferenceCode *= 16
			t.characterReferenceCode += int(char - '7')
			return nil
		}

		t.characterReferenceCode *= 16
		t.characterReferenceCode += int(char - '\u0057')
	}

	if char == ';' {
		t.state = state_NumericCharacterReferenceEnd
		return nil
	}

	t.state = state_NumericCharacterReferenceEnd
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#decimal-character-reference-state
func (t *Tokenizer) state_DeciamalCharacterReference() error {
	char, err := t.consume()
	if err != nil {
		return err
	}

	if unicode.IsDigit(char) {
		t.characterReferenceCode *= 10
		t.characterReferenceCode += int(char - '0')
	}

	if char == ';' {
		t.state = state_NumericCharacterReferenceEnd
		return nil
	}

	t.state = state_NumericCharacterReferenceEnd
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#numeric-character-reference-end-state
func (t *Tokenizer) state_NumericCharacterReferenceEnd() error {

	if t.characterReferenceCode == 0x00 {
		t.characterReferenceCode = 0xFFFD
	}

	if t.characterReferenceCode > 0x10FFFF {
		t.characterReferenceCode = 0xFFFD
	}

	if (t.characterReferenceCode >= 0xD800 && t.characterReferenceCode <= 0xDBFF) ||
		(t.characterReferenceCode >= 0xCD00 && t.characterReferenceCode <= 0xDFFF) {
		t.characterReferenceCode = 0xFFFD
	}

	//  noncharacter-character-reference parser error
	/*if (t.characterReferenceCode >= 0xFDD0 && t.characterReferenceCode <= 0xFDEF) ||
		slices.Contains([]int{
			0xFFFE, 0xFFFF, 0x1FFFE, 0x1FFFF, 0x2FFFE, 0x2FFFF, 0x3FFFE, 0x3FFFF, 0x4FFFE, 0x4FFFF, 0x5FFFE, 0x5FFFF,
			0x6FFFE, 0x6FFFF, 0x7FFFE, 0x7FFFF, 0x8FFFE, 0x8FFFF, 0x9FFFE, 0x9FFFF, 0xAFFFE, 0xAFFFF, 0xBFFFE,
			0xBFFFF, 0xCFFFE, 0xCFFFF, 0xDFFFE, 0xDFFFF, 0xEFFFE, 0xEFFFF, 0xFFFFE, 0xFFFFF, 0x10FFFE, 0x10FFFF,
		}, t.characterReferenceCode) {

	}*/
	if t.characterReferenceCode == 0x0D || (t.characterReferenceCode >= 0x0000 && t.characterReferenceCode <= 0x001F) {
		switch t.characterReferenceCode {
		case 0x80:
			t.characterReferenceCode = 0x20AC
		case 0x82:
			t.characterReferenceCode = 0x201A
		case 0x83:
			t.characterReferenceCode = 0x0192
		case 0x84:
			t.characterReferenceCode = 0x201E
		case 0x85:
			t.characterReferenceCode = 0x2026
		case 0x86:
			t.characterReferenceCode = 0x2020
		case 0x87:
			t.characterReferenceCode = 0x2021
		case 0x88:
			t.characterReferenceCode = 0x02C6
		case 0x89:
			t.characterReferenceCode = 0x2030
		case 0x8A:
			t.characterReferenceCode = 0x0160
		case 0x8B:
			t.characterReferenceCode = 0x2039
		case 0x8C:
			t.characterReferenceCode = 0x0152
		case 0x8E:
			t.characterReferenceCode = 0x017D
		case 0x91:
			t.characterReferenceCode = 0x2018
		case 0x92:
			t.characterReferenceCode = 0x2019
		case 0x93:
			t.characterReferenceCode = 0x201C
		case 0x94:
			t.characterReferenceCode = 0x201D
		case 0x95:
			t.characterReferenceCode = 0x2022
		case 0x96:
			t.characterReferenceCode = 0x2013
		case 0x97:
			t.characterReferenceCode = 0x2014
		case 0x98:
			t.characterReferenceCode = 0x02DC
		case 0x99:
			t.characterReferenceCode = 0x2122
		case 0x9A:
			t.characterReferenceCode = 0x0161
		case 0x9B:
			t.characterReferenceCode = 0x203A
		case 0x9C:
			t.characterReferenceCode = 0x0153
		case 0x9E:
			t.characterReferenceCode = 0x017E
		case 0x9F:
			t.characterReferenceCode = 0x0178
		}

	}

	t.tempbuffer = string(rune(t.characterReferenceCode))

	t.flush()
	t.state = t.rstate.MustSome()
	return nil
}

//#endregion
