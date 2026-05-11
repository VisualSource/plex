package html_tokenizer

import (
	"io"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/runeio"
	"github.com/VisualSource/plex/internal/utils"
)

// https://html.spec.whatwg.org/#tokenization
type Tokenizer struct {
	reader                 *runeio.RuneReader
	state                  TokenizerState
	rstate                 TokenizerState
	tokens                 []Token
	characterReferenceCode int
	tempbuffer             strings.Builder
	workingToken           Token
	lastStartTag           utils.StringOption
	errors                 []TokenizerError

	workingAttrName  strings.Builder
	workingAttrValue strings.Builder

	adjustedNodeIsNotHTML bool
}

func (t *Tokenizer) SetAdjustedNodeIsNotHTML(v bool) {
	t.adjustedNodeIsNotHTML = v
}

func NewTokenizer(stream io.Reader) *Tokenizer {
	return &Tokenizer{
		reader:       runeio.NewReader(stream),
		state:        State_Data,
		rstate:       state_Unset,
		lastStartTag: utils.None[string](),
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

func (t *Tokenizer) SetState(state TokenizerState) {
	t.state = state
}

func (t *Tokenizer) ReconsumeToken(token Token) {
	t.tokens = slices.Insert(t.tokens, 0, token)
}

func (t *Tokenizer) HasEmittedTokens() bool {
	return len(t.tokens) != 0
}

func (t *Tokenizer) ConsumeToken() Token {
	if !t.HasEmittedTokens() {
		return nil
	}

	x, a := t.tokens[0], t.tokens[1:]
	t.tokens = a

	return x
}

//#region internal utils

// When a state says to flush code points consumed as a character reference,
// it means that for each code point in the temporary buffer (in the order they were added to the buffer),
// the user agent must append the code point from the buffer to the current attribute's value if the character reference
// was consumed as part of an attribute, or emit the code point as a character token otherwise.
func (t *Tokenizer) flush() error {
	if wasConsumedAsPartOfAttribute(t.rstate) {
		_, err := t.workingAttrValue.WriteString(t.tempbuffer.String())
		return err
	}

	for _, char := range t.tempbuffer.String() {
		t.tokens = append(t.tokens, NewTokenCharacter(char))
	}

	return nil
}

// https://html.spec.whatwg.org/#preprocessing-the-input-stream
// https://infra.spec.whatwg.org/#normalize-newlines
func (t *Tokenizer) consume() (rune, error) {
	rune, _, err := t.reader.ReadRune()
	if err != nil {
		return utf8.RuneError, err
	}

	if rune != '\r' {
		if unicode.IsControl(rune) && !(IsWhitespace(rune) || rune == '\u0000') {
			t.errors = append(t.errors, NewTokenizerError(ErrControlCharacterInInputStream, -1, -1))
		}
		if isNonCharacterCodepoint(int(rune)) {
			t.errors = append(t.errors, NewTokenizerError(ErrNoncharacterInInputStream, -1, -1))
		}
		if isSurrogate(int(rune)) {
			t.errors = append(t.errors, NewTokenizerError(ErrSurrogateInInputStream, -1, -1))
		}

		return rune, nil
	}

	// Remove CRLF (\r\n) line endings

	// if the next run is a '\n' return a '\n' for the '\r\n'
	// if its a EOF we can ignore it as it should be handle by the consumer on next call
	// if none conditions above don't apply, unread the call below and replace '\r' with '\n'
	v, _, err := t.reader.ReadRune()
	if err == io.EOF || v == '\n' {
		return '\n', nil
	}

	if err = t.reader.UnreadRune(); err != nil {
		return utf8.RuneError, err
	}

	return '\n', nil
}

func (t *Tokenizer) finishAttr() {

	if tag, ok := t.workingToken.(*TokenTag); ok {
		if tag.t == TokenEndTag {
			t.errors = append(t.errors, NewTokenizerError(ErrEndTagWithAttributes, -1, -1))
		}

		attrName := t.workingAttrName.String()

		if attrName == "" {
			return
		}

		_, ok := tag.Attributes[attrName]
		if !ok {
			tag.Attributes[attrName] = dom.NewAttribute(dom.NamespaceHTML, attrName, t.workingAttrValue.String())
		} else {
			t.errors = append(t.errors, NewTokenizerError(ErrDuplicateAttribute, -1, -1))
		}
	}

	t.workingAttrName.Reset()
	t.workingAttrValue.Reset()
}

func (t *Tokenizer) emitCurrentWithTokens(tokens ...Token) {
	t.finishAttr()
	if tag, ok := t.workingToken.(*TokenTag); ok {
		switch tag.t {
		case TokenEndTag:
			if tag.selfClosing.SomeOr(false) {
				t.errors = append(t.errors, NewTokenizerError(ErrEndTagWithTrailingSolidus, -1, -1))
			}

		case TokenStartTag:
			t.lastStartTag.Set(tag.name)
		}
	}

	if t.workingToken != nil {
		t.tokens = append(t.tokens, t.workingToken)
	}

	t.tokens = append(t.tokens, tokens...)
	t.workingToken = nil
}

// An appropriate end tag token is an end tag token whose tag name matches the tag name of the last start tag to have been
// emitted from this tokenizer, if any. If no start tag has been emitted from this tokenizer,
// then no end tag token is appropriate.
func (t *Tokenizer) hasApproriateEndTagToken() bool {
	tag, ok := t.workingToken.(*TokenTag)

	if !ok || tag.t == TokenStartTag || t.lastStartTag.IsNone() {
		return false
	}

	return t.lastStartTag.Is(tag.name)
}

//#endregion

//#region Entry

// https://html.spec.whatwg.org/#data-state
func (t *Tokenizer) state_Data() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.tokens = append(t.tokens, NewTokenEOF())
		}
		return err
	}

	if char == '&' {
		t.rstate = t.state
		t.state = state_CharacterReference
		return nil
	}

	if char == '<' {
		t.state = state_TagOpen
		return nil
	}

	if char == '\u0000' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
	}

	t.tokens = append(t.tokens, NewTokenCharacter(char))

	return nil
}

// https://html.spec.whatwg.org/#rcdata-state
func (t *Tokenizer) state_RCData() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.tokens = append(t.tokens, NewTokenEOF())
			return io.EOF
		}
		return err
	}

	if char == '&' {
		t.rstate = State_RCData
		t.state = state_CharacterReference
		return nil
	}

	if char == '<' {
		t.state = state_RCData_LessThanSign
		return nil
	}

	if char == '\u0000' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter(char))

	return nil
}

// https://html.spec.whatwg.org/#rawtext-state
func (t *Tokenizer) state_RawText() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.tokens = append(t.tokens, NewTokenEOF())
			return io.EOF
		}
		return err
	}

	if char == '<' {
		t.state = state_RawText_LessThanSign
		return nil
	}

	if char == '\u0000' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter(char))

	return nil
}

// https://html.spec.whatwg.org/#script-data-state
func (t *Tokenizer) state_ScriptData() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.tokens = append(t.tokens, NewTokenEOF())
			return io.EOF
		}
		return err
	}

	if char == '<' {
		t.state = state_ScriptData_LessThanSign
		return nil
	}

	if char == '\u0000' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter(char))

	return nil
}

// https://html.spec.whatwg.org/#plaintext-state
func (t *Tokenizer) state_PlainText() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.tokens = append(t.tokens, NewTokenEOF())
		}
		return err
	}

	if char == '\u0000' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
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

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofBeforeTagName, -1, -1))
			t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenEOF())
		}
		return err
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
		t.workingToken = NewTokenTag("", TokenStartTag, utils.None[bool]())
		return t.reader.UnreadRune()
	}

	if char == '?' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedQuestionMarkInsteadOfTagName, -1, -1))
		t.workingToken = NewTokenComment("")
		t.state = state_BogusComment
		return t.reader.UnreadRune()
	}

	t.errors = append(t.errors, NewTokenizerError(ErrInvalidFirstCharacterOfTagName, -1, -1))
	t.tokens = append(t.tokens, NewTokenCharacter('<'))
	t.state = State_Data
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#end-tag-open-state
func (t *Tokenizer) state_EndTagOpen() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofBeforeTagName, -1, -1))
			t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('/'), NewTokenEOF())
		}

		return err
	}

	if unicode.IsLetter(char) {
		t.workingToken = NewTokenTag("", TokenEndTag, utils.None[bool]())
		t.state = state_TagName
		return t.reader.UnreadRune()
	}

	if char == '>' {
		t.errors = append(t.errors, NewTokenizerError(ErrMissingEndTagName, -1, -1))
		t.state = State_Data
		return nil
	}

	t.errors = append(t.errors, NewTokenizerError(ErrInvalidFirstCharacterOfTagName, -1, -1))
	t.workingToken = NewTokenComment("")

	t.state = state_BogusComment
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#tag-name-state
func (t *Tokenizer) state_TagName() error {
	char, err := t.consume()

	switch {
	case err != nil:
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInTag, -1, -1))
			t.tokens = append(t.tokens, NewTokenEOF())
		}
		return err
	case IsWhitespace(char):
		t.state = state_BeforeAttributeName
	case char == '/':
		t.state = state_SelfClosingStartTag
	case char == '>':
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
	case unicode.IsLetter(char) && unicode.IsUpper(char):
		if tag, ok := t.workingToken.(*TokenTag); ok {
			tag.name += string(unicode.ToLower(char))
		}
	case char == '\u0000':
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		if tag, ok := t.workingToken.(*TokenTag); ok {
			tag.name += string(utf8.RuneError)
		}
	default:
		if tag, ok := t.workingToken.(*TokenTag); ok {
			tag.name += string(char)
		}
	}

	return nil
}

//#endregion

//#region RCDATA

// https://html.spec.whatwg.org/#rcdata-less-than-sign-state
func (t *Tokenizer) state_RCData_LessThen() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return err
	}

	if char == '/' {
		t.tempbuffer.Reset()
		t.state = state_RCData_EndTagOpen
		return nil
	}

	t.state = State_RCData
	t.tokens = append(t.tokens, NewTokenCharacter('<'))

	if err == io.EOF {
		return nil
	}

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#rcdata-end-tag-open-state
func (t *Tokenizer) state_RCData_EndTagOpen() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return err
	}

	if unicode.IsLetter(char) {
		t.workingToken = NewTokenTag("", TokenEndTag, utils.None[bool]())

		t.state = state_RCData_EndTagName
		return t.reader.UnreadRune()
	}

	t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('/'))
	t.state = State_RCData

	if err == io.EOF {
		return nil
	}

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#rcdata-end-tag-name-state
func (t *Tokenizer) state_RCData_EndTagName() error {
	char, err := t.consume()

	switch {
	case err != nil && err != io.EOF:
		return err
	case IsWhitespace(char) && t.hasApproriateEndTagToken():
		t.state = state_BeforeAttributeName
	case char == '/' && t.hasApproriateEndTagToken():
		t.state = state_SelfClosingStartTag
	case char == '>' && t.hasApproriateEndTagToken():
		t.state = State_Data
		t.emitCurrentWithTokens()
		t.reader.Forget()
	case unicode.IsLetter(char):
		t.tempbuffer.WriteRune(char)
		if tag, ok := t.workingToken.(*TokenTag); ok {
			if unicode.IsUpper(char) {
				tag.name += string(unicode.ToLower(char))
			} else {
				tag.name += string(char)
			}
		}
	default:
		t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('/'))
		for _, char := range t.tempbuffer.String() {
			t.tokens = append(t.tokens, NewTokenCharacter(char))
		}

		t.state = State_RCData
		if err == io.EOF {
			return nil
		}
		return t.reader.UnreadRune()
	}

	return nil
}

//#endregion

//#region RAWTEXT

// https://html.spec.whatwg.org/#rawtext-less-than-sign-state
func (t *Tokenizer) state_RawText_LessThenSign() error {
	char, err := t.consume()

	if err != nil && err != io.EOF {
		return err
	}

	if char == '/' {
		t.tempbuffer.Reset()
		t.state = state_RawText_EndTagOpen
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter('<'))
	t.state = State_RawText

	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#rawtext-end-tag-open-state
func (t *Tokenizer) state_RawText_EndTagOpen() error {
	char, err := t.consume()

	if err != nil && err != io.EOF {
		return err
	}

	if unicode.IsLetter(char) {
		t.workingToken = NewTokenTag("", TokenEndTag, utils.None[bool]())
		t.state = state_RawText_EndTagName

		return t.reader.UnreadRune()
	}

	t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('/'))
	t.state = State_RawText

	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#rawtext-end-tag-name-state
func (t *Tokenizer) state_RawText_EndTagName() error {
	char, err := t.consume()

	switch {
	case err != nil && err != io.EOF:
		return err
	case IsWhitespace(char) && t.hasApproriateEndTagToken():
		t.state = state_BeforeAttributeName
	case char == '/' && t.hasApproriateEndTagToken():
		t.state = state_SelfClosingStartTag
	case char == '>' && t.hasApproriateEndTagToken():
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
	case unicode.IsLetter(char):
		t.tempbuffer.WriteRune(char)
		if tag, ok := t.workingToken.(*TokenTag); ok {
			if unicode.IsUpper(char) {
				tag.name += string(unicode.ToLower(char))
			} else {
				tag.name += string(char)
			}
		}
	default:
		t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('/'))
		for _, char := range t.tempbuffer.String() {
			t.tokens = append(t.tokens, NewTokenCharacter(char))
		}

		t.state = State_RawText
		if err == io.EOF {
			return nil
		}
		return t.reader.UnreadRune()
	}

	return nil
}

//#endregion

// #region Script
// https://html.spec.whatwg.org/#script-data-less-than-sign-state
func (t *Tokenizer) state_ScriptData_LessThanSign() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return err
	}

	switch char {
	case '/':
		t.tempbuffer.Reset()
		t.state = state_ScriptData_EndTagOpen
		return nil
	case '!':
		t.state = state_ScriptData_EscapeStart
		t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('!'))
		return nil
	default:
		t.tokens = append(t.tokens, NewTokenCharacter('<'))
		t.state = State_ScriptData
		if err == io.EOF {
			return nil
		}
		return t.reader.UnreadRune()
	}
}

// https://html.spec.whatwg.org/#script-data-end-tag-open-state
func (t *Tokenizer) state_ScriptData_EndTagOpen() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return nil
	}

	if unicode.IsLetter(char) {
		t.workingToken = NewTokenTag("", TokenEndTag, utils.None[bool]())
		t.state = state_ScriptData_EndTagName
		return t.reader.UnreadRune()
	}

	t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('/'))
	t.state = State_ScriptData

	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-end-tag-name-state
func (t *Tokenizer) state_ScriptData_EndTagName() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return nil
	}

	switch {
	case IsWhitespace(char) && t.hasApproriateEndTagToken():
		t.state = state_BeforeAttributeName
		return nil
	case char == '/' && t.hasApproriateEndTagToken():
		t.state = state_SelfClosingStartTag
		return nil
	case char == '>' && t.hasApproriateEndTagToken():
		t.state = State_Data
		t.emitCurrentWithTokens()
		t.reader.Forget()
		return nil
	case unicode.IsLetter(char):
		t.tempbuffer.WriteRune(char)
		if tag, ok := t.workingToken.(*TokenTag); ok {
			if unicode.IsUpper(char) {
				tag.name += string(unicode.ToLower(char))
			} else {
				tag.name += string(char)
			}
		}
		return nil
	default:
		t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('/'))
		for _, char := range t.tempbuffer.String() {
			t.tokens = append(t.tokens, NewTokenCharacter(char))
		}

		t.state = State_ScriptData
		if err == io.EOF {
			return nil
		}
		return t.reader.UnreadRune()
	}
}

// https://html.spec.whatwg.org/#script-data-escape-start-state
func (t *Tokenizer) state_ScriptData_Escape_Start() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return nil
	}

	if char == '-' {
		t.state = state_ScriptData_EscapeStartDash
		t.tokens = append(t.tokens, NewTokenCharacter('-'))
		return nil
	}

	t.state = State_ScriptData
	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-escape-start-dash-state
func (t *Tokenizer) state_ScriptData_Escape_StartDash() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return nil
	}

	if char == '-' {
		t.state = state_ScriptData_EscapedDashDash
		t.tokens = append(t.tokens, NewTokenCharacter('-'))
		return nil
	}

	t.state = State_ScriptData
	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-escaped-state
func (t *Tokenizer) state_ScriptData_Escaped() error {
	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInScriptHtmlCommentLikeText, -1, -1))
			t.tokens = append(t.tokens, NewTokenEOF())
		}
		return err
	}

	switch char {
	case '-':
		t.state = state_ScriptData_EscapedDash
		t.tokens = append(t.tokens, NewTokenCharacter('-'))
		return nil
	case '<':
		t.state = state_ScriptData_EscapedLessThanSign
		return nil
	case '\u0000':
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	default:
		t.tokens = append(t.tokens, NewTokenCharacter(char))
		return nil
	}
}

// https://html.spec.whatwg.org/#script-data-escaped-dash-state
func (t *Tokenizer) state_ScriptData_Escaped_Dash() error {
	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInScriptHtmlCommentLikeText, -1, -1))
			t.tokens = append(t.tokens, NewTokenEOF())
		}
		return err
	}

	switch char {
	case '-':
		t.state = state_ScriptData_EscapedDashDash
		t.tokens = append(t.tokens, NewTokenCharacter('-'))
		return nil
	case '<':
		t.state = state_ScriptData_EscapedLessThanSign
		return nil
	case '\u0000':
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		t.state = state_ScriptData_Escaped
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	default:
		t.state = state_ScriptData_Escaped
		t.tokens = append(t.tokens, NewTokenCharacter(char))
		return nil
	}
}

// https://html.spec.whatwg.org/#script-data-escaped-dash-dash-state
func (t *Tokenizer) state_ScriptData_Escaped_DashDash() error {
	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInScriptHtmlCommentLikeText, -1, -1))
			t.tokens = append(t.tokens, NewTokenEOF())
		}
		return err
	}

	switch char {
	case '-':
		t.tokens = append(t.tokens, NewTokenCharacter('-'))
		return nil
	case '<':
		t.state = state_ScriptData_EscapedLessThanSign
		return nil
	case '>':
		t.state = State_ScriptData
		t.tokens = append(t.tokens, NewTokenCharacter('>'))
		return nil
	case '\u0000':
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		t.state = state_ScriptData_Escaped
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	default:
		t.state = state_ScriptData_Escaped
		t.tokens = append(t.tokens, NewTokenCharacter(char))
		return nil
	}
}

// https://html.spec.whatwg.org/#script-data-escaped-less-than-sign-state
func (t *Tokenizer) state_ScriptData_Escaped_LessThanSign() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return err
	}

	if char == '/' {
		t.tempbuffer.Reset()
		t.state = state_ScriptData_EscapedEndTagOpen
		return nil
	}

	if unicode.IsLetter(char) {
		t.tempbuffer.Reset()
		t.tokens = append(t.tokens, NewTokenCharacter('<'))
		t.state = state_ScriptData_DoubleEscapedStart
		return t.reader.UnreadRune()
	}

	t.tokens = append(t.tokens, NewTokenCharacter('<'))
	t.state = state_ScriptData_Escaped
	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-escaped-end-tag-open-state
func (t *Tokenizer) state_ScriptData_Escaped_EndTagOpen() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return nil
	}

	if unicode.IsLetter(char) {
		t.workingToken = NewTokenTag("", TokenEndTag, utils.None[bool]())
		t.state = state_ScriptData_EscapedEndTagName
		return t.reader.UnreadRune()
	}

	t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('/'))
	t.state = state_ScriptData_Escaped
	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-escaped-end-tag-name-state
func (t *Tokenizer) state_ScriptData_Escaped_EndTagName() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return nil
	}

	if IsWhitespace(char) && t.hasApproriateEndTagToken() {
		t.state = state_BeforeAttributeName
		return nil
	}

	if char == '/' && t.hasApproriateEndTagToken() {
		t.state = state_SelfClosingStartTag
		return nil
	}

	if char == '>' && t.hasApproriateEndTagToken() {
		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if unicode.IsLetter(char) {
		t.tempbuffer.WriteRune(char)
		if tag, ok := t.workingToken.(*TokenTag); ok {
			if unicode.IsUpper(char) {
				tag.name += string(unicode.ToLower(char))
			} else {
				tag.name += string(char)
			}
		}
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter('<'), NewTokenCharacter('/'))

	for _, char := range t.tempbuffer.String() {
		t.tokens = append(t.tokens, NewTokenCharacter(char))
	}

	t.state = state_ScriptData_Escaped
	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-double-escape-start-state
func (t *Tokenizer) state_ScriptData_DoubleEscapedStart() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return nil
	}

	if IsWhitespace(char) || char == '/' || char == '>' {
		if t.tempbuffer.String() == "script" {
			t.state = state_ScriptData_DoubleEscaped
		} else {
			t.state = state_ScriptData_Escaped
		}

		t.tokens = append(t.tokens, NewTokenCharacter(char))
		return nil
	}

	if unicode.IsLetter(char) {
		if unicode.IsUpper(char) {
			t.tempbuffer.WriteRune(unicode.ToLower(char))
		} else {
			t.tempbuffer.WriteRune(char)
		}

		t.tokens = append(t.tokens, NewTokenCharacter(char))
		return nil
	}

	t.state = state_ScriptData_Escaped
	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-double-escaped-state
func (t *Tokenizer) state_ScriptData_DoubleEscaped() error {
	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInScriptHtmlCommentLikeText, -1, -1))
			t.tokens = append(t.tokens, NewTokenEOF())
		}
		return err
	}

	switch char {
	case '-':
		t.state = state_ScriptData_DoubleEscapedDash
		t.tokens = append(t.tokens, NewTokenCharacter('-'))
		return nil
	case '<':
		t.state = state_ScriptData_DoubleEscapedLessThanSign
		t.tokens = append(t.tokens, NewTokenCharacter('<'))
		return nil
	case '\u0000':
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	default:
		t.tokens = append(t.tokens, NewTokenCharacter(char))
		return nil
	}
}

// https://html.spec.whatwg.org/#script-data-double-escaped-dash-state
func (t *Tokenizer) state_ScriptData_DoubleEscapedDash() error {
	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInScriptHtmlCommentLikeText, -1, -1))
			t.tokens = append(t.tokens, NewTokenEOF())
		}
		return err
	}

	switch char {
	case '-':
		t.state = state_ScriptData_DoubleEscapedDashDash
		t.tokens = append(t.tokens, NewTokenCharacter('-'))
		return nil
	case '<':
		t.state = state_ScriptData_DoubleEscapedLessThanSign
		t.tokens = append(t.tokens, NewTokenCharacter('<'))
		return nil
	case '\u0000':
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		t.state = state_ScriptData_DoubleEscaped
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	default:
		t.state = state_ScriptData_DoubleEscaped
		t.tokens = append(t.tokens, NewTokenCharacter(char))
		return nil
	}
}

// https://html.spec.whatwg.org/#script-data-double-escaped-dash-dash-state
func (t *Tokenizer) state_ScriptData_DoubleEscapedDashDash() error {
	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInScriptHtmlCommentLikeText, -1, -1))
			t.tokens = append(t.tokens, NewTokenEOF())
		}
		return err
	}

	switch char {
	case '-':
		t.tokens = append(t.tokens, NewTokenCharacter('-'))
		return nil
	case '<':
		t.tokens = append(t.tokens, NewTokenCharacter('<'))
		t.state = state_ScriptData_DoubleEscapedLessThanSign
		return nil
	case '>':
		t.state = State_ScriptData
		t.tokens = append(t.tokens, NewTokenCharacter('>'))
		return nil
	case '\u0000':
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		t.state = state_ScriptData_DoubleEscaped
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	default:
		t.state = state_ScriptData_DoubleEscaped
		t.tokens = append(t.tokens, NewTokenCharacter(char))
		return nil
	}
}

// https://html.spec.whatwg.org/#script-data-double-escaped-less-than-sign-state
func (t *Tokenizer) state_ScriptData_DoubleEscaped_LessThanSign() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return nil
	}

	if char == '/' {
		t.tempbuffer.Reset()
		t.state = state_ScriptData_DoubleEscapeEnd
		t.emitCurrentWithTokens(NewTokenCharacter('/'))
		t.reader.Forget()
		return nil
	}

	t.state = state_ScriptData_DoubleEscaped
	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#script-data-double-escape-end-state
func (t *Tokenizer) state_ScriptData_DoubleEscapedEnd() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return nil
	}

	if IsWhitespace(char) || char == '/' || char == '>' {
		if t.tempbuffer.String() == "script" {
			t.state = state_ScriptData_Escaped
		} else {
			t.state = state_ScriptData_DoubleEscaped
		}

		t.emitCurrentWithTokens(NewTokenCharacter(char))
		t.reader.Forget()
		return nil
	}

	if unicode.IsLetter(char) {
		if unicode.IsUpper(char) {
			t.tempbuffer.WriteRune(unicode.ToLower(char))
		} else {
			t.tempbuffer.WriteRune(char)
		}

		t.emitCurrentWithTokens(NewTokenCharacter(char))
		t.reader.Forget()
		return nil
	}

	t.state = state_ScriptData_DoubleEscaped
	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

//#endregion
//#region Attribute

// https://html.spec.whatwg.org/#before-attribute-name-state
func (t *Tokenizer) state_BeforeAttributeName() error {
	char, err := t.consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF || char == '/' || char == '>' {
		t.state = state_AfterAttributeName
		if isEOF {
			return nil // can not unread eof
		}
		return t.reader.UnreadRune()
	}

	if IsWhitespace(char) {
		return nil
	}

	if char == '=' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedEqualsSignBeforeAttributeName, -1, -1))
		t.state = state_AttributeName

		t.finishAttr()
		t.workingAttrName.WriteRune(char)
		return nil
	}

	t.finishAttr()

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

	if isEOF || IsWhitespace(char) || char == '/' || char == '>' {
		t.state = state_AfterAttributeName

		if isEOF {
			return nil
		}

		return t.reader.UnreadRune()
	}

	if char == '=' {
		t.state = state_BeforeAttributeValue
		return nil
	}

	if unicode.IsLetter(char) && unicode.IsUpper(char) {
		t.workingAttrName.WriteRune(unicode.ToLower(char))
		return nil
	}

	if char == '\u0000' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		t.workingAttrName.WriteRune(utf8.RuneError)
		return nil
	}

	if char == '"' || char == '\'' || char == '<' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedCharacterInAttributeName, -1, -1))
	}

	t.workingAttrName.WriteRune(char)

	return nil
}

// https://html.spec.whatwg.org/#after-attribute-name-state
func (t *Tokenizer) state_AfterAttributeName() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInTag, -1, -1))
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if IsWhitespace(char) {
		return nil
	}

	switch char {
	case '/':
		t.state = state_SelfClosingStartTag
		return nil
	case '=':
		t.state = state_BeforeAttributeValue
		return nil
	case '>':
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	default:
		t.finishAttr()

		t.state = state_AttributeName
		return t.reader.UnreadRune()
	}
}

// https://html.spec.whatwg.org/#before-attribute-value-state
func (t *Tokenizer) state_BeforeAttributeValue() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return err
	}

	if IsWhitespace(char) {
		return nil
	}

	switch char {
	case '"':
		t.state = state_AttributValue_DoubleQuoted
		return nil
	case '\'':
		t.state = state_AttributValue_SingleQuoted
		return nil
	case '>':
		t.errors = append(t.errors, NewTokenizerError(ErrMissingAttributeValue, -1, -1))
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	default:
		t.state = state_AttributValue_Unquoted
		if err == io.EOF {
			return nil
		}
		return t.reader.UnreadRune()
	}
}

// https://html.spec.whatwg.org/#attribute-value-(double-quoted)-state
func (t *Tokenizer) state_AttributeValue_DoubleQuote() error {
	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInTag, -1, -1))
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	switch char {
	case '"':
		t.state = state_AfterAttributeValue_Quoted
		return nil
	case '&':
		t.rstate = state_AttributValue_DoubleQuoted
		t.state = state_CharacterReference
		return nil
	case '\u0000':
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		t.workingAttrValue.WriteRune(utf8.RuneError)
		return nil
	default:
		t.workingAttrValue.WriteRune(char)
		return nil
	}
}

// https://html.spec.whatwg.org/#attribute-value-(single-quoted)-state
func (t *Tokenizer) state_AttributeValue_SignleQuote() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInTag, -1, -1))
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	switch char {
	case '\'':
		t.state = state_AfterAttributeValue_Quoted
		return nil
	case '&':
		t.rstate = state_AttributValue_SingleQuoted
		t.state = state_CharacterReference
		return nil
	case '\u0000':
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		t.workingAttrValue.WriteRune(utf8.RuneError)
		return nil
	default:
		t.workingAttrValue.WriteRune(char)
		return nil
	}
}

// https://html.spec.whatwg.org/#attribute-value-(unquoted)-state
func (t *Tokenizer) state_AttributeValue_Unquoted() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInTag, -1, -1))
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if IsWhitespace(char) {
		t.state = state_BeforeAttributeName
		return nil
	}

	switch char {
	case '&':
		t.rstate = state_AttributValue_Unquoted
		t.state = state_CharacterReference
		return nil
	case '>':
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	case '\u0000':
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		t.workingAttrValue.WriteRune(utf8.RuneError)
		return nil
	case '"', '\'', '<', '=', '`':
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedCharacterInUnquotedAttributeValue, -1, -1))
		fallthrough
	default:
		t.workingAttrValue.WriteRune(char)
		return nil
	}
}

// https://html.spec.whatwg.org/#after-attribute-value-(quoted)-state
func (t *Tokenizer) state_AfterAttributeValue_Quoted() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInTag, -1, -1))
			t.tokens = append(t.tokens, NewTokenEOF())
		}
		return err
	}

	if IsWhitespace(char) {
		t.state = state_BeforeAttributeName
		return nil
	}

	if char == '/' {
		t.state = state_SelfClosingStartTag
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	}

	t.errors = append(t.errors, NewTokenizerError(ErrMissingWhitespaceBetweenAttributes, -1, -1))
	t.state = state_BeforeAttributeName
	return t.reader.UnreadRune()
}

//#endregion
//#endregion

// https://html.spec.whatwg.org/#self-closing-start-tag-state
func (t *Tokenizer) state_SelfClosingStartTag() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInTag, -1, -1))
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if char == '>' {
		if tag, ok := t.workingToken.(*TokenTag); ok {
			tag.selfClosing.Set(true)
		}
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	}

	t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedSolidusInTag, -1, -1))

	t.state = state_BeforeAttributeName
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#markup-declaration-open-state
func (t *Tokenizer) state_MarkupDeclarationOpen() error {
	runes, err := t.reader.Peek(7)
	nonFilledPeek := err == io.EOF
	if err != nil && !nonFilledPeek {
		return err
	}

	if len(runes) >= 2 && runes[0] == '-' && runes[1] == '-' {
		out := make([]rune, 2)
		_, err := t.reader.Read(out)
		if err != nil {
			return err
		}

		t.state = state_CommentStart
		t.workingToken = NewTokenComment("")
		return nil
	}

	if len(runes) == 7 {
		value := string(runes)

		if strings.ToUpper(value) == "DOCTYPE" {
			out := make([]rune, 7)
			_, err := t.reader.Read(out)
			if err != nil {
				return err
			}
			t.state = state_DOCTYPE
			return nil
		} else if value == "[CDATA[" {
			out := make([]rune, 7)
			_, err := t.reader.Read(out)
			if err != nil {
				return err
			}

			if t.adjustedNodeIsNotHTML {
				t.state = state_CDATA_Section
				return nil
			}

			t.errors = append(t.errors, NewTokenizerError(ErrCdataInHtmlContent, -1, -1))
			t.workingToken = NewTokenComment(value)
			t.state = state_BogusComment
			return nil
		}
	}

	t.errors = append(t.errors, NewTokenizerError(ErrIncorrectlyOpenedComment, -1, -1))
	t.state = state_BogusComment
	t.workingToken = NewTokenComment("")
	return nil
}

//#endregion
//#region Comment

// https://html.spec.whatwg.org/#bogus-comment-state
func (t *Tokenizer) state_BogusComment() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if char == '>' {
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	}

	if char == '\u0000' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		if tag, ok := t.workingToken.(*TokenComment); ok {
			tag.Value += string(utf8.RuneError)
		}
		return nil
	}

	if tag, ok := t.workingToken.(*TokenComment); ok {
		tag.Value += string(char)
	}
	return nil
}

// https://html.spec.whatwg.org/#comment-start-state
func (t *Tokenizer) state_CommentStart() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return err
	}

	if char == '-' {
		t.state = state_CommentStartDash
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.reader.Forget()
		t.errors = append(t.errors, NewTokenizerError(ErrAbruptClosingOfEmptyComment, -1, -1))
		t.emitCurrentWithTokens()
		return nil
	}

	t.state = state_Comment
	if err != io.EOF {
		return t.reader.UnreadRune()
	}

	return nil
}

// https://html.spec.whatwg.org/#comment-start-dash-state
func (t *Tokenizer) state_Comment_StartDash() error {
	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInComment, -1, -1))
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	switch char {
	case '-':
		t.state = state_CommentEnd
		return nil
	case '>':
		t.state = State_Data
		t.reader.Forget()
		t.errors = append(t.errors, NewTokenizerError(ErrAbruptClosingOfEmptyComment, -1, -1))
		t.emitCurrentWithTokens()
		return nil
	default:
		t.state = state_Comment
		if tag, ok := t.workingToken.(*TokenComment); ok {
			tag.Value += "-"
		}
		return t.reader.UnreadRune()
	}
}

// https://html.spec.whatwg.org/#comment-state
func (t *Tokenizer) state_Comment() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInComment, -1, -1))
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	switch char {
	case '<':
		t.state = state_CommentLessThanSign
		if tag, ok := t.workingToken.(*TokenComment); ok {
			tag.Value += string(char)
		}
		return nil
	case '-':
		t.state = state_CommentEndDash
		return nil
	case '\u0000':
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		if tag, ok := t.workingToken.(*TokenComment); ok {
			tag.Value += string(utf8.RuneError)
		}
		return nil
	default:
		if tag, ok := t.workingToken.(*TokenComment); ok {
			tag.Value += string(char)
		}
		return nil
	}
}

// https://html.spec.whatwg.org/#comment-less-than-sign-state
func (t *Tokenizer) state_Comment_LessThanSign() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return err
	}

	switch char {
	case '!':
		t.state = state_CommentLessThanSignBang
		if tag, ok := t.workingToken.(*TokenComment); ok {
			tag.Value += string(char)
		}
		return nil
	case '<':
		if tag, ok := t.workingToken.(*TokenComment); ok {
			tag.Value += string(char)
		}
		return nil
	default:
		t.state = state_Comment
		if err == io.EOF {
			return nil
		}
		return t.reader.UnreadRune()
	}
}

// https://html.spec.whatwg.org/#comment-less-than-sign-bang-state
func (t *Tokenizer) state_Comment_LessThanSign_Bang() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return err
	}

	if char == '-' {
		t.state = state_CommentLessThanSignBangDash
		return nil
	}

	t.state = state_Comment

	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#comment-less-than-sign-bang-dash-state
func (t *Tokenizer) state_Comment_LessThanSign_BangDash() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return err
	}

	if char == '-' {
		t.state = state_CommentLessThanSignBangDashDash
		return nil
	}

	t.state = state_CommentEndDash
	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#comment-less-than-sign-bang-dash-dash-state
func (t *Tokenizer) state_Comment_LessThanSign_BangDashDash() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if !isEOF && err != nil {
		return err
	}

	t.state = state_CommentEnd

	if !(isEOF || char == '>') {
		t.errors = append(t.errors, NewTokenizerError(ErrNestedComment, -1, -1))
	}

	if isEOF {
		// spec calls for "to reconsume" but due to reader impl
		// attempting to call 'UnreadRune' when the previous read call returned an EOF error
		// when result in a panic of case out of bounds index
		//
		// so return nil and let the "Comment End" state handle EOF
		return nil
	}

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#comment-end-dash-state
func (t *Tokenizer) state_Comment_EndDash() error {
	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInComment, -1, -1))
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if char == '-' {
		t.state = state_CommentEnd
		return nil
	}

	t.state = state_Comment
	if tag, ok := t.workingToken.(*TokenComment); ok {
		tag.Value += "-"
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#comment-end-state
func (t *Tokenizer) state_CommentEnd() error {
	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInComment, -1, -1))
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	switch char {
	case '>':
		t.reader.Forget()

		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	case '!':
		t.state = state_CommentEndBang
		return nil
	case '-':
		if tag, ok := t.workingToken.(*TokenComment); ok {
			tag.Value += string(char)
		}
		return nil
	default:
		t.state = state_Comment
		if tag, ok := t.workingToken.(*TokenComment); ok {
			tag.Value += "--"
		}
		return t.reader.UnreadRune()
	}

}

// https://html.spec.whatwg.org/#comment-end-bang-state
func (t *Tokenizer) state_Comment_EndBang() error {
	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInComment, -1, -1))
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	switch char {
	case '-':
		t.state = state_CommentEndDash
		if tag, ok := t.workingToken.(*TokenComment); ok {
			tag.Value += "--!"
		}
		return nil
	case '>':
		t.errors = append(t.errors, NewTokenizerError(ErrIncorrectlyClosedComment, -1, -1))
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	default:
		t.state = state_Comment
		if tag, ok := t.workingToken.(*TokenComment); ok {
			tag.Value += "--!"
		}
		return t.reader.UnreadRune()
	}
}

//#endregion

//#region DOCTYPE

// https://html.spec.whatwg.org/#doctype-state
func (t *Tokenizer) state_DOCTYPE() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			tag := NewTokenDOCTYPE(utils.None[string](), utils.None[string](), utils.None[string](), true)
			t.emitCurrentWithTokens(tag, NewTokenEOF())
			return err
		}
		return err
	}

	if IsWhitespace(char) {
		t.state = state_BeforeDOCTYPEName
		return nil
	}

	if char == '>' {
		t.state = state_BeforeDOCTYPEName
		return t.reader.UnreadRune()
	}

	t.errors = append(t.errors, NewTokenizerError(ErrMissingWhitespaceBeforeDoctypeName, -1, -1))
	t.state = state_BeforeDOCTYPEName
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#before-doctype-name-state
func (t *Tokenizer) state_BeforeDOCTYPEName() error {
	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			tag := NewTokenDOCTYPE(utils.None[string](), utils.None[string](), utils.None[string](), true)
			t.emitCurrentWithTokens(tag, NewTokenEOF())
		}
		return err
	}

	if IsWhitespace(char) {
		return nil
	}

	if unicode.IsLetter(char) && unicode.IsUpper(char) {
		t.workingToken = NewTokenDOCTYPE(utils.Some(string(unicode.ToLower(char))), utils.None[string](), utils.None[string](), false)
		t.state = state_DOCTYPE_Name
		return nil
	}

	if char == '\u0000' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		t.workingToken = NewTokenDOCTYPE(utils.Some(string(utf8.RuneError)), utils.None[string](), utils.None[string](), false)
		t.state = state_DOCTYPE_Name
		return nil
	}

	if char == '>' {
		t.errors = append(t.errors, NewTokenizerError(ErrMissingDoctypeName, -1, -1))
		t.workingToken = NewTokenDOCTYPE(utils.None[string](), utils.None[string](), utils.None[string](), true)
		t.state = State_Data

		t.reader.Forget()

		t.emitCurrentWithTokens()

		return nil
	}

	t.workingToken = NewTokenDOCTYPE(utils.Some(string(char)), utils.None[string](), utils.None[string](), false)
	t.state = state_DOCTYPE_Name

	return nil
}

// https://html.spec.whatwg.org/#doctype-name-state
func (t *Tokenizer) state_DOCTYPE_Name() error {
	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
				tag.forceQuirks = true
			}
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if IsWhitespace(char) {
		t.state = state_AfterDOCTYPE_Name
		return nil
	}

	if char == '>' {
		t.state = State_Data

		t.reader.Forget()

		t.emitCurrentWithTokens()
		return nil
	}

	if unicode.IsLetter(char) && unicode.IsUpper(char) {
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok && tag.name.IsSome() {
			*tag.name.Value += string(unicode.ToLower(char))
		}
		return nil
	}

	if char == '\u0000' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok && tag.name.IsSome() {
			*tag.name.Value += string(utf8.RuneError)
		}
		return nil
	}
	if tag, ok := t.workingToken.(*TokenDOCTYPE); ok && tag.name.IsSome() {
		*tag.name.Value += string(char)
	}

	return nil
}

// https://html.spec.whatwg.org/#after-doctype-name-state
func (t *Tokenizer) state_AfterDOCTYPE_Name() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
				tag.forceQuirks = true
			}
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if IsWhitespace(char) {
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.reader.Forget()
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

	t.errors = append(t.errors, NewTokenizerError(ErrInvalidCharacterSequenceAfterDoctypeName, -1, -1))
	t.state = state_BogusDOCTYPE
	if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
		tag.forceQuirks = true
	}

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#after-doctype-public-keyword-state
func (t *Tokenizer) state_AfterDOCTYPE_PublicKeyword() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
				tag.forceQuirks = true
			}
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if IsWhitespace(char) {
		t.state = state_BeforeDOCTYPE_PublicIdentifier
		return nil
	}

	if char == '"' {
		t.errors = append(t.errors, NewTokenizerError(ErrMissingWhitespaceAfterDoctypePublicKeyword, -1, -1))
		t.state = state_DOCTYPE_PublicIdentifier_DoubleQuoted
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.publicIdentifier = utils.Some("")
		}
		return nil
	}

	if char == '\'' {
		t.errors = append(t.errors, NewTokenizerError(ErrMissingWhitespaceAfterDoctypePublicKeyword, -1, -1))
		t.state = state_DOCTYPE_PublicIdentifier_SingleQuoted
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.publicIdentifier = utils.Some("")
		}
		return nil
	}

	if char == '>' {
		t.errors = append(t.errors, NewTokenizerError(ErrMissingDoctypePublicIdentifier, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.state = State_Data

		t.reader.Forget()

		t.emitCurrentWithTokens()
		return nil
	}

	if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
		tag.forceQuirks = true
	}

	t.errors = append(t.errors, NewTokenizerError(ErrMissingQuoteBeforeDoctypePublicIdentifer, -1, -1))
	t.state = state_BogusDOCTYPE
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#before-doctype-public-identifier-state
func (t *Tokenizer) state_BeforeDOCTYPE_PublicIdentifier() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
				tag.forceQuirks = true
			}
			t.emitCurrentWithTokens(NewTokenEOF())
			t.state = State_Data
		}
		return err
	}

	if IsWhitespace(char) {
		return nil
	}

	if char == '"' {
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.publicIdentifier = utils.Some("")
		}
		t.state = state_DOCTYPE_PublicIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.publicIdentifier = utils.Some("")
		}
		t.state = state_DOCTYPE_PublicIdentifier_SingleQuoted
		return nil
	}

	if char == '>' {
		t.errors = append(t.errors, NewTokenizerError(ErrMissingDoctypePublicIdentifier, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}

		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	}

	if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
		tag.forceQuirks = true
	}
	t.errors = append(t.errors, NewTokenizerError(ErrMissingQuoteBeforeDoctypePublicIdentifer, -1, -1))
	t.state = state_BogusDOCTYPE
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#doctype-public-identifier-(double-quoted)-state
func (t *Tokenizer) state_DOCTYPE_PublicIdentifier_DoubleQuoted() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
				tag.forceQuirks = true
			}
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if char == '"' {
		t.state = state_AfterDOCTYPE_PublicIdentifier
		return nil
	}

	if char == '\u0000' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok && tag.publicIdentifier.IsSome() {
			*tag.publicIdentifier.Value += string(utf8.RuneError)
		}
		return nil
	}

	if char == '>' {
		t.errors = append(t.errors, NewTokenizerError(ErrAbruptDoctypePublicIdentifer, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	}
	if tag, ok := t.workingToken.(*TokenDOCTYPE); ok && tag.publicIdentifier.IsSome() {
		*tag.publicIdentifier.Value += string(char)
	}

	return nil
}

// https://html.spec.whatwg.org/#doctype-public-identifier-(single-quoted)-state
func (t *Tokenizer) state_DOCKTYPE_PublicIdentifier_SingleQuoted() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
				tag.forceQuirks = true
			}
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if char == '\'' {
		t.state = state_AfterDOCTYPE_PublicIdentifier
		return nil
	}

	if char == '\u0000' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok && tag.publicIdentifier.IsSome() {
			*tag.publicIdentifier.Value += string(utf8.RuneError)
		}
		return nil
	}

	if char == '>' {
		t.errors = append(t.errors, NewTokenizerError(ErrAbruptDoctypePublicIdentifer, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.state = State_Data
		t.reader.Forget()

		t.emitCurrentWithTokens()
		return nil
	}
	if tag, ok := t.workingToken.(*TokenDOCTYPE); ok && tag.publicIdentifier.IsSome() {
		*tag.publicIdentifier.Value += string(char)
	}

	return nil
}

// https://html.spec.whatwg.org/#after-doctype-public-identifier-state
func (t *Tokenizer) state_AfterDOCTYPE_PublicIdentifier() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
				tag.forceQuirks = true
			}
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if IsWhitespace(char) {
		t.state = state_BetweenDOCTYPE_PublicAndSystemIdentifiers
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	}

	if char == '"' {
		t.errors = append(t.errors, NewTokenizerError(ErrNoWSBetweenDoctypePublicAndSystenIdentifier, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.systemIdentifier = utils.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		t.errors = append(t.errors, NewTokenizerError(ErrNoWSBetweenDoctypePublicAndSystenIdentifier, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.systemIdentifier = utils.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_SingleQuoted
		return nil
	}

	if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
		tag.forceQuirks = true
	}

	t.errors = append(t.errors, NewTokenizerError(ErrMissingQuoteBeforeDoctypeSystemIdentifier, -1, -1))
	t.state = state_BogusDOCTYPE

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#between-doctype-public-and-system-identifiers-state
func (t *Tokenizer) state_BetweenDOCTYPE_PublicAndSystemIdent() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
				tag.forceQuirks = true
			}
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if IsWhitespace(char) {
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	}

	if char == '"' {
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.systemIdentifier = utils.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.systemIdentifier = utils.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_SingleQuoted
		return nil
	}

	t.errors = append(t.errors, NewTokenizerError(ErrMissingQuoteBeforeDoctypeSystemIdentifier, -1, -1))
	if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
		tag.forceQuirks = true
	}
	t.state = state_BogusDOCTYPE

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#after-doctype-system-keyword-state
func (t *Tokenizer) state_AfterDOCTYPE_SystemKeyword() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
				tag.forceQuirks = true
			}
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if IsWhitespace(char) {
		return nil
	}

	if char == '"' {
		t.errors = append(t.errors, NewTokenizerError(ErrMissingWhitespaceAfterDoctypeSystemKeyword, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.systemIdentifier = utils.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		t.errors = append(t.errors, NewTokenizerError(ErrMissingWhitespaceAfterDoctypeSystemKeyword, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.systemIdentifier = utils.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_SingleQuoted
		return nil
	}

	if char == '>' {
		t.errors = append(t.errors, NewTokenizerError(ErrMissingDoctypeSystemIdentifier, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	}

	t.errors = append(t.errors, NewTokenizerError(ErrMissingQuoteBeforeDoctypeSystemIdentifier, -1, -1))
	if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
		tag.forceQuirks = true
	}
	t.state = state_BogusDOCTYPE

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#before-doctype-system-identifier-state
func (t *Tokenizer) state_BeforeDOCTYPE_SystemIdentifier() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
				tag.forceQuirks = true
			}
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if IsWhitespace(char) {
		return nil
	}

	if char == '"' {
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.systemIdentifier = utils.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.systemIdentifier = utils.Some("")
		}
		t.state = state_DOCTYPE_SystemIdentifier_SingleQuoted
		return nil
	}

	if char == '>' {
		t.errors = append(t.errors, NewTokenizerError(ErrMissingDoctypeSystemIdentifier, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	}

	t.errors = append(t.errors, NewTokenizerError(ErrMissingQuoteBeforeDoctypeSystemIdentifier, -1, -1))
	if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
		tag.forceQuirks = true
	}
	t.state = state_BogusDOCTYPE

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#doctype-system-identifier-(double-quoted)-state
func (t *Tokenizer) state_DOCTYPE_SystemIdentifier_DoubleQuoted() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
				tag.forceQuirks = true
			}
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if char == '"' {
		t.state = state_AfterDOCTYPE_SystemIdentifier
		return nil
	}

	if char == '\u0000' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok && tag.systemIdentifier.IsSome() {
			*tag.systemIdentifier.Value += string(utf8.RuneError)
		}
		return nil
	}

	if char == '>' {
		t.errors = append(t.errors, NewTokenizerError(ErrAbruptDoctypeSystemIdentifier, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	}

	if tag, ok := t.workingToken.(*TokenDOCTYPE); ok && tag.systemIdentifier.IsSome() {
		*tag.systemIdentifier.Value += string(char)
	}

	return nil
}

// https://html.spec.whatwg.org/#doctype-system-identifier-(single-quoted)-state
func (t *Tokenizer) state_DOCTYPE_SystemIdentifier_SingleQuoted() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
				tag.forceQuirks = true
			}
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if char == '\'' {
		t.state = state_AfterDOCTYPE_SystemIdentifier
		return nil
	}

	if char == '\u0000' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok && tag.systemIdentifier.IsSome() {
			*tag.systemIdentifier.Value += string(utf8.RuneError)
		}
		return nil
	}

	if char == '>' {
		t.errors = append(t.errors, NewTokenizerError(ErrAbruptDoctypeSystemIdentifier, -1, -1))
		if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
			tag.forceQuirks = true
		}
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	}

	if tag, ok := t.workingToken.(*TokenDOCTYPE); ok && tag.systemIdentifier.IsSome() {
		*tag.systemIdentifier.Value += string(char)
	}

	return nil
}

// https://html.spec.whatwg.org/#after-doctype-system-identifier-state
func (t *Tokenizer) state_AfterDOCTYPE_SystemIdentifer() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInDoctype, -1, -1))
			if tag, ok := t.workingToken.(*TokenDOCTYPE); ok {
				tag.forceQuirks = true
			}
			t.emitCurrentWithTokens(NewTokenEOF())
		}
		return err
	}

	if IsWhitespace(char) {
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.reader.Forget()
		t.emitCurrentWithTokens()
		return nil
	}

	t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedCharAfterDoctypeSystemIdentifier, -1, -1))
	t.state = state_BogusDOCTYPE
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#bogus-doctype-state
func (t *Tokenizer) state_Bogus_DOCTYPE() error {
	char, err := t.consume()

	if err != nil {
		if err == io.EOF {
			t.emitCurrentWithTokens(NewTokenEOF())

		}
		return err
	}

	if char == '>' {
		t.reader.Forget()

		t.state = State_Data
		t.emitCurrentWithTokens()
		return nil
	}

	if char == '\u0000' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnexpectedNullCharacter, -1, -1))
	}

	return nil
}

//#endregion

//#region CDATA

// https://html.spec.whatwg.org/#cdata-section-state
func (t *Tokenizer) state_CDATA_Section() error {
	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			t.errors = append(t.errors, NewTokenizerError(ErrEofInCDATA, -1, -1))
			t.tokens = append(t.tokens, NewTokenEOF())
		}
		return err
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
	if err != nil && err != io.EOF {
		return err
	}

	if char == ']' {
		t.state = state_CDATA_SectionEnd
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter(']'))
	t.state = state_CDATA_Section
	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#cdata-section-end-state
func (t *Tokenizer) state_CDATA_SectionEnd() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return err
	}

	if char == ']' {
		t.tokens = append(t.tokens, NewTokenCharacter(']'))
		return nil
	}

	if char == '>' {
		t.reader.Forget()
		t.state = State_Data
		return nil
	}

	t.tokens = append(t.tokens, NewTokenCharacter(']'), NewTokenCharacter(']'))
	t.state = state_CDATA_Section
	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

//#endregion

//#region reference

// https://html.spec.whatwg.org/#character-reference-state
func (t *Tokenizer) state_CharacterReference() error {
	t.tempbuffer.Reset()
	t.tempbuffer.WriteRune('&')

	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			err = t.flush()
			t.state = t.rstate
			return err
		}
		return err
	}

	if unicode.IsNumber(char) || unicode.IsLetter(char) {
		t.state = state_NamedCharacterReference
		return t.reader.UnreadRune()
	}

	if char == '#' {
		t.tempbuffer.WriteRune('#')
		t.state = state_NumericCharacterReference
		return nil
	}

	if err := t.flush(); err != nil {
		return err
	}
	t.state = t.rstate
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#named-character-reference-state
func (t *Tokenizer) state_NamedCharacterReference() error {

	runes, err := t.reader.Peek(33)
	if err != nil && err != io.EOF {
		return err
	}

	codepoints, size := getCharacterReference(&runes)
	if codepoints == nil {
		t.state = state_AmbiguousAmpersand
		return t.flush()
	}

	out := make([]rune, size)
	if _, err = t.reader.Read(out); err != nil {
		return err
	}

	if _, err := t.tempbuffer.WriteString(string(out)); err != nil {
		return err
	}

	lastIdx := size - 1
	if wasConsumedAsPartOfAttribute(t.rstate) && runes[lastIdx] != ';' && (runes[size] == '=' || unicode.IsDigit(runes[size]) || unicode.IsLetter(runes[size])) {
		err = t.flush()
		t.reader.Forget()
		t.state = t.rstate
		return err
	}

	if runes[lastIdx] != ';' {
		t.errors = append(t.errors, NewTokenizerError(ErrMissingSemicolonAfterCharacterReference, -1, -1))
	}

	t.tempbuffer.Reset()
	for _, codepoint := range codepoints {
		t.tempbuffer.WriteRune(rune(codepoint))
	}

	err = t.flush()
	t.reader.Forget()
	t.state = t.rstate

	return err
}

// https://html.spec.whatwg.org/#ambiguous-ampersand-state
func (t *Tokenizer) state_AmbiguousAmpersand() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return err
	}

	if unicode.IsLetter(char) || unicode.IsNumber(char) {
		if wasConsumedAsPartOfAttribute(t.rstate) {
			t.workingAttrValue.WriteRune(char)
			return nil
		}

		t.tokens = append(t.tokens, NewTokenCharacter(char))
		return nil
	}

	if char == ';' {
		t.errors = append(t.errors, NewTokenizerError(ErrUnknownNamedCharacterReference, -1, -1))
	}

	t.state = t.rstate
	if err == io.EOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#numeric-character-reference-state
func (t *Tokenizer) state_NumericCharacterReference() error {
	t.characterReferenceCode = 0

	char, err := t.consume()
	if err != nil {
		if err == io.EOF {
			// "anything else" case
			// don't call unreadRune do to indexing panic
			t.state = state_DecimalCharacterReferenceStart
			return nil
		}
		return err
	}

	if char == 'x' || char == 'X' {
		t.tempbuffer.WriteRune(char)
		t.state = state_HexadecimalCharacterReferenceStart
		return nil
	}

	t.state = state_DecimalCharacterReferenceStart
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#hexadecimal-character-reference-start-state
func (t *Tokenizer) state_HexadecimalCharacterReferenceStart() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if !isEOF && err != nil {
		return err
	}

	if !isEOF && unicode.Is(unicode.ASCII_Hex_Digit, char) {
		t.state = state_HexadecimalCharacterReference
		return t.reader.UnreadRune()
	}

	t.errors = append(t.errors, NewTokenizerError(ErrAbsenceOfDigitsInNumericCharacterReference, -1, -1))
	if err := t.flush(); err != nil {
		return err
	}
	t.state = t.rstate
	if isEOF {
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#hexadecimal-character-reference-start-state
func (t *Tokenizer) state_DecimalCharacterReferenceStart() error {
	char, err := t.consume()
	isEOF := err == io.EOF
	if !isEOF && err != nil {
		return err
	}

	if !isEOF && unicode.IsDigit(char) {
		t.state = state_DecimalCharacterReference
		return t.reader.UnreadRune()
	}

	t.errors = append(t.errors, NewTokenizerError(ErrAbsenceOfDigitsInNumericCharacterReference, -1, -1))
	if err := t.flush(); err != nil {
		return err
	}
	t.state = t.rstate
	if isEOF {
		// don't unread when EOF was returned
		return nil
	}
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#hexadecimal-character-reference-state
func (t *Tokenizer) state_HexadecimalCharacterReference() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return err
	}

	if unicode.IsDigit(char) {
		t.characterReferenceCode = addNumericDigit(int(char-'0'), 16, t.characterReferenceCode)
		return nil
	}

	if unicode.Is(unicode.ASCII_Hex_Digit, char) {
		if unicode.IsUpper(char) {
			t.characterReferenceCode = addNumericDigit(int(char-'7'), 16, t.characterReferenceCode)
			return nil
		}

		t.characterReferenceCode = addNumericDigit(int(char-'W'), 16, t.characterReferenceCode)
		return nil
	}

	if char == ';' {
		t.state = state_NumericCharacterReferenceEnd
		return nil
	}

	t.errors = append(t.errors, NewTokenizerError(ErrMissingSemicolonAfterCharacterReference, -1, -1))
	t.state = state_NumericCharacterReferenceEnd

	if err == io.EOF {
		return nil
	}

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#decimal-character-reference-state
func (t *Tokenizer) state_DeciamalCharacterReference() error {
	char, err := t.consume()
	if err != nil && err != io.EOF {
		return err
	}

	if unicode.IsDigit(char) {
		t.characterReferenceCode = addNumericDigit(int(char-'0'), 10, t.characterReferenceCode)
		return nil
	}

	if char == ';' {
		t.state = state_NumericCharacterReferenceEnd
		return nil
	}

	t.errors = append(t.errors, NewTokenizerError(ErrMissingSemicolonAfterCharacterReference, -1, -1))
	t.state = state_NumericCharacterReferenceEnd

	if err == io.EOF {
		return nil
	}

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#numeric-character-reference-end-state
func (t *Tokenizer) state_NumericCharacterReferenceEnd() error {

	if t.characterReferenceCode == 0x00 {
		t.errors = append(t.errors, NewTokenizerError(ErrNullCharacterReference, -1, -1))
		t.characterReferenceCode = utf8.RuneError
	}

	if t.characterReferenceCode > 0x10FFFF {
		t.errors = append(t.errors, NewTokenizerError(ErrCharacterReferenceOutsideUnicodeRange, -1, -1))
		t.characterReferenceCode = utf8.RuneError
	}

	if isSurrogate(t.characterReferenceCode) {
		t.errors = append(t.errors, NewTokenizerError(ErrSurrogateCharacterReference, -1, -1))
		t.characterReferenceCode = utf8.RuneError
	}

	if isNonCharacterCodepoint(t.characterReferenceCode) {
		t.errors = append(t.errors, NewTokenizerError(ErrNoncharacterCharacterReference, -1, -1))
	}

	if t.characterReferenceCode == 0x0D ||
		(t.characterReferenceCode >= 0x0000 && t.characterReferenceCode <= 0x001F) ||
		(t.characterReferenceCode >= 0x007F && t.characterReferenceCode <= 0x009F && !IsWhitespace(rune(t.characterReferenceCode))) {
		t.errors = append(t.errors, NewTokenizerError(ErrControlCharacterReference, -1, -1))
		switch t.characterReferenceCode {
		case 0x80:
			t.characterReferenceCode = 0x20AC //EURO SIGN
		case 0x82:
			t.characterReferenceCode = 0x201A //SINGLE LOW-9 QUOTATION MARK
		case 0x83:
			t.characterReferenceCode = 0x0192 //LATIN SMALL LETTER F WITH HOOK
		case 0x84:
			t.characterReferenceCode = 0x201E //DOUBLE LOW-9 QUOTATION MARK
		case 0x85:
			t.characterReferenceCode = 0x2026 //HORIZONTAL ELLIPSIS
		case 0x86:
			t.characterReferenceCode = 0x2020 //DAGGER
		case 0x87:
			t.characterReferenceCode = 0x2021 //DOUBLE DAGGER
		case 0x88:
			t.characterReferenceCode = 0x02C6 //MODIFIER LETTER CIRCUMFLEX ACCENT
		case 0x89:
			t.characterReferenceCode = 0x2030 //PER MILLE SIGN
		case 0x8A:
			t.characterReferenceCode = 0x0160 //LATIN CAPITAL LETTER S WITH CARON
		case 0x8B:
			t.characterReferenceCode = 0x2039 //SINGLE LEFT-POINTING ANGLE QUOTATION MARK
		case 0x8C:
			t.characterReferenceCode = 0x0152 //LATIN CAPITAL LIGATURE OE
		case 0x8E:
			t.characterReferenceCode = 0x017D //LATIN CAPITAL LETTER Z WITH CARON
		case 0x91:
			t.characterReferenceCode = 0x2018 //LEFT SINGLE QUOTATION MARK
		case 0x92:
			t.characterReferenceCode = 0x2019 //RIGHT SINGLE QUOTATION MARK
		case 0x93:
			t.characterReferenceCode = 0x201C //LEFT DOUBLE QUOTATION MARK
		case 0x94:
			t.characterReferenceCode = 0x201D //RIGHT DOUBLE QUOTATION MARK
		case 0x95:
			t.characterReferenceCode = 0x2022 //BULLET
		case 0x96:
			t.characterReferenceCode = 0x2013 //EN DASH
		case 0x97:
			t.characterReferenceCode = 0x2014 //EM DASH
		case 0x98:
			t.characterReferenceCode = 0x02DC //SMALL TILDE
		case 0x99:
			t.characterReferenceCode = 0x2122 //TRADE MARK SIGN
		case 0x9A:
			t.characterReferenceCode = 0x0161 //LATIN SMALL LETTER S WITH CARON
		case 0x9B:
			t.characterReferenceCode = 0x203A //SINGLE RIGHT-POINTING ANGLE QUOTATION MARK
		case 0x9C:
			t.characterReferenceCode = 0x0153 //LATIN SMALL LIGATURE OE
		case 0x9E:
			t.characterReferenceCode = 0x017E //LATION SMALL LETTER Z WITH CARON
		case 0x9F:
			t.characterReferenceCode = 0x0178 //LATIN CAPITAL LETTER Y WITH DIAERESIS
		}
	}

	t.tempbuffer.Reset()
	t.tempbuffer.WriteRune(rune(t.characterReferenceCode))

	err := t.flush()
	t.reader.Forget()
	t.state = t.rstate
	return err
}

//#endregion
