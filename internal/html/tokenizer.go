package html

import (
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ianlewis/runeio"
)

const (
	Token_EOF = iota
	Token_Char
	Token_StartTag
	Token_Comment
	Token_EndTag
	Token_DOCTYPE
)

const (
	State_CharacterReference = iota
	State_TagOpen
	State_RCData
	State_RCData_LessThanSign
	State_RawText_LessThanSign
	State_ScriptData_LessThanSign
	State_MarkupDeclarationOpen
	State_EndTagOpen
	State_TagName
	State_BogusComment
	State_Data
	State_BeforeAttributeName
	State_SelfClosingStartTag
	State_RCData_EndTagOpen
	State_RCData_EndTagName
	State_RawText_EndTagOpen
	State_RawText
	State_RawText_EndTagName
	State_ScriptData_EndTagOpen
	State_ScriptData_EscapeStart
	State_ScriptData
	State_ScriptData_EndTagName
	State_ScriptData_EscapeStartDash
	State_ScriptData_EscapedDashDash
	State_ScriptData_EscapedDash
	State_ScriptData_EscapedLessThanSign
	State_ScriptData_Escaped
	State_ScriptData_EscapedEndTagOpen
	State_ScriptData_DoubleEscapeStart
	State_ScriptData_EscapedEndTagName
	State_ScriptData_DoubleEscaped
	State_ScriptData_DoubleEscapedDash
	State_ScriptData_DoubleEscapedLessThanSign
	State_ScriptData_DoubleEscapedDashDash
	State_ScriptData_DoubleEscapeEnd
	State_AfterAttributeName
	State_AttributeName
	State_AttributValue_DoubleQuoted
	State_AttributValue_SingleQuoted
	State_AttributValue_Unquoted
	State_AfterAttributeValue_Quoted
	State_CommentStart
	State_DOCTYPE
	State_CDATA_Section
	State_CommentStartDash
	State_Comment
	State_CommentEnd
	State_CommentLessThanSign
	State_CommentEndDash
	State_CommentLessThanSignBang
	State_CommentLessThanSignBangDash
	State_CommentLessThanSignBangDashDash
	State_CommentEndBang
	State_BeforeDOCTYPEName
	State_DOCTYPE_Name
	State_AfterDOCTYPE_PublicKeyword
	State_AfterDOCTYPE_SystemKeyword
	State_BogusDOCTYPE
	State_AfterDOCTYPE_Name
	State_BeforeDOCTYPE_PublicIDentifier
	State_DOCTYPE_PublicIdentifier_DoubleQuoted
	State_DOCTYPE_PublicIdentifier_SingleQuoted
	State_AfterDOCTYPE_PublicIdentifier
	State_BetweenDOCTYPE_PublicAndSystemIdentifiers
	State_DOCTYPE_SystemIdentifier_DoubleQuoted
	State_DOCTYPE_SystemIdentifier_SingleQuoted
	State_AfterDOCTYPE_SystemIdentifier
	State_CDATA_SectionBracket
	State_CDATA_SectionEnd
	State_NamedCharacterReference
	State_NumericCharacterReference
	State_AmbiguousAmpersand
	State_HexadeciamCharacterReferenceStart
	State_DecimalCharacterReferenceStart
	State_NumericCharacterReferenceEnd
)

type Token struct {
	token       int
	char        rune
	value       string
	selfClosing bool
	state       map[string]string
}

func NewEOFToken() Token {
	return Token{token: Token_EOF}
}

func NewReplacementToken() Token {
	return Token{token: Token_Char, char: '\ufffd'}
}

func NewCharToken(value rune) Token {
	return Token{token: Token_Char, char: value}
}

// https://html.spec.whatwg.org/#tokenization
type Tokenizer struct {
	reader runeio.RuneReader
	state  int
	rstate int
	tokens []Token

	tempbuffer string

	workingToken *Token
	cAttrName    string
	cAttrValue   string
}

func (t *Tokenizer) Parse() ([]Token, error) {

	var err error = nil
	for {
		switch t.state {
		case State_CharacterReference:
			err = t.CharacterReferenceState()
		case State_TagOpen:
			err = t.TagOpenState()
		case State_RCData:
			err = t.RCDataState()
		case State_RCData_LessThanSign:
			err = t.RCData_LessThenState()

		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	return t.tokens, nil
}

func (t *Tokenizer) Consume() (rune, error) {
	r, _, err := t.reader.ReadRune()

	if err != nil {
		return r, err
	}

	return r, nil
}

func (t *Tokenizer) HasApproriateEndTagToken() bool {
	return false
}

func (t *Tokenizer) DataState() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return io.EOF
	}

	if char == '&' {
		t.rstate = t.state
		t.state = State_CharacterReference
		return nil
	}

	if char == '<' {
		t.state = State_TagOpen
		return nil
	}

	if char == '\u0000' {
		t.tokens = append(t.tokens, NewCharToken(char))
		return nil
	}

	t.tokens = append(t.tokens, NewCharToken(char))

	return nil
}
func (t *Tokenizer) RCDataState() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return io.EOF
	}

	if char == '&' {
		t.rstate = State_RCData
		t.state = State_CharacterReference
		return nil
	}

	if char == '<' {
		t.state = State_RCData_LessThanSign
		return nil
	}

	if char == '\u0000' {
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewCharToken(char))

	return nil
}
func (t *Tokenizer) RawTextState() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return io.EOF
	}

	if char == '<' {
		t.state = State_RawText_LessThanSign
		return nil
	}

	if char == '\u0000' {
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewCharToken(char))

	return nil
}
func (t *Tokenizer) ScriptDataState() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return io.EOF
	}

	if char == '<' {
		t.state = State_ScriptData_LessThanSign
		return nil
	}

	if char == '\u0000' {
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewCharToken(char))

	return nil
}
func (t *Tokenizer) PlainTextState() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return nil
	}

	if char == '\u0000' {
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewCharToken(char))

	return nil
}
func (t *Tokenizer) TagOpenState() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewCharToken('<'), NewEOFToken())
		return io.EOF
	}

	if char == '!' {
		t.state = State_MarkupDeclarationOpen
		return nil
	}

	if char == '/' {
		t.state = State_EndTagOpen
		return nil
	}

	if unicode.IsLetter(char) {
		err = t.reader.UnreadRune()
		if err != nil {
			return err
		}
		t.state = State_TagName

		t.workingToken = &Token{token: Token_StartTag, value: ""}

		return nil
	}

	if char == '?' {
		err = t.reader.UnreadRune()
		if err != nil {
			return err
		}
		t.state = State_BogusComment

		t.workingToken = &Token{token: Token_Comment, value: ""}

		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.tokens = append(t.tokens, NewCharToken('>'))
	t.state = State_Data

	return nil
}
func (t *Tokenizer) EndTagOpenState() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewCharToken('<'), NewCharToken('/'), NewEOFToken())
		return io.EOF
	}

	if char == '>' {
		t.state = State_Data
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_BogusComment

	return nil
}

// https://html.spec.whatwg.org/#tag-name-state
func (t *Tokenizer) TagNameState() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return io.EOF
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == ' ' {
		t.state = State_BeforeAttributeName
		return nil
	}

	if char == '/' {
		t.state = State_SelfClosingStartTag
		return nil
	}

	if char == '>' {
		t.state = State_Data

		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil

		return nil
	}

	if unicode.IsLetter(char) && unicode.IsUpper(char) {
		t.workingToken.value += string(unicode.ToLower(char))

		return nil
	}

	if char == '\u0000' {
		t.workingToken.value += string('\ufffd')
		return nil
	}

	t.workingToken.value += string(char)
	return nil
}

// https://html.spec.whatwg.org/#rcdata-less-than-sign-state
func (t *Tokenizer) RCData_LessThenState() error {
	char, err := t.Consume()

	if err != nil {
		return err
	}

	if char == '/' {
		t.tempbuffer = ""
		t.state = State_RCData_EndTagOpen
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.tokens = append(t.tokens, NewCharToken('<'))

	return nil
}

// https://html.spec.whatwg.org/#rcdata-end-tag-open-state
func (t *Tokenizer) RCData_EndTagOpenState() error {
	char, err := t.Consume()

	if err != nil {
		return err
	}

	if unicode.IsLetter(char) {
		err = t.reader.UnreadRune()
		if err != nil {
			return err
		}
		t.workingToken = &Token{token: Token_EndTag, value: ""}
		t.state = State_RCData_EndTagName

		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.tokens = append(t.tokens, NewCharToken('<'), NewCharToken('/'))
	t.state = State_RCData

	return nil
}

// https://html.spec.whatwg.org/#rcdata-end-tag-name-state
func (t *Tokenizer) RCData_EndTagNameState() error {
	char, err := t.Consume()

	if err != nil {
		return err
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		if t.HasApproriateEndTagToken() {
			t.state = State_BeforeAttributeName
			return nil
		}
	}

	if char == '/' {
		if t.HasApproriateEndTagToken() {
			t.state = State_SelfClosingStartTag
			return nil
		}
	}

	if char == '>' {
		if t.HasApproriateEndTagToken() {
			t.state = State_Data

			t.tokens = append(t.tokens, *t.workingToken)
			t.workingToken = nil

			return nil
		}
	}

	if unicode.IsLetter(char) {
		if unicode.IsUpper(char) {

			t.workingToken.value += string(unicode.ToLower(char))
			t.tempbuffer += string(char)

			return nil
		}

		t.workingToken.value += string(char)
		t.tempbuffer += string(char)

		return nil
	}

	t.tokens = append(t.tokens, NewCharToken('<'), NewCharToken('/'))

	for x := range t.tempbuffer {
		t.tokens = append(t.tokens, NewCharToken(rune(t.tempbuffer[x])))
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_RCData

	return nil
}

// https://html.spec.whatwg.org/#rawtext-less-than-sign-state
func (t *Tokenizer) RawText_LessThenSignState() error {
	char, err := t.Consume()

	if err != nil {
		return err
	}

	if char == '/' {
		t.tempbuffer = ""
		t.state = State_RawText_EndTagOpen
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.tokens = append(t.tokens, NewCharToken('<'))
	t.state = State_RawText

	return nil
}

// https://html.spec.whatwg.org/#rawtext-end-tag-open-state
func (t *Tokenizer) RawText_EndTagOpenState() error {
	char, err := t.Consume()

	if err != nil {
		return err
	}

	if unicode.IsLetter(char) {
		err = t.reader.UnreadRune()
		if err != nil {
			return err
		}
		t.workingToken = &Token{token: Token_EndTag, value: ""}
		t.state = State_RawText_EndTagName

		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}

	t.tokens = append(t.tokens, NewCharToken('<'), NewCharToken('/'))
	t.state = State_RawText

	return nil
}

// https://html.spec.whatwg.org/#rawtext-end-tag-name-state
func (t *Tokenizer) RawText_EndTagNameState() error {
	char, err := t.Consume()

	if err != nil {
		return nil
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		if t.HasApproriateEndTagToken() {
			t.state = State_BeforeAttributeName
			return nil
		}
	}

	if char == '/' {
		if t.HasApproriateEndTagToken() {
			t.state = State_SelfClosingStartTag
			return nil
		}
	}

	if char == '>' {
		if t.HasApproriateEndTagToken() {
			t.state = State_Data
			t.tokens = append(t.tokens, *t.workingToken)
			t.workingToken = nil
			return nil
		}
	}

	if unicode.IsLetter(char) {
		if unicode.IsUpper(char) {
			t.workingToken.value += string(unicode.ToLower(char))
			t.tempbuffer += string(char)
			return nil
		}

		t.workingToken.value += string(char)
		t.tempbuffer += string(char)

		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.tokens = append(t.tokens, NewCharToken('<'), NewCharToken('/'))
	t.state = State_RawText

	for x := range t.tempbuffer {
		t.tokens = append(t.tokens, NewCharToken(rune(t.tempbuffer[x])))
	}

	return nil
}

// https://html.spec.whatwg.org/#script-data-less-than-sign-state
func (t *Tokenizer) ScriptData_LessThanSignState() error {
	char, err := t.Consume()

	if err != nil {
		return err
	}

	if char == '/' {
		t.tempbuffer = ""
		t.state = State_ScriptData_EndTagOpen
		return nil
	}

	if char == '!' {
		t.state = State_ScriptData_EscapeStart
		t.tokens = append(t.tokens, NewCharToken('<'), NewCharToken('!'))
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.tokens = append(t.tokens, NewCharToken('<'))
	t.state = State_ScriptData

	return nil
}

// https://html.spec.whatwg.org/#script-data-end-tag-open-state
func (t *Tokenizer) ScriptData_EndTagOpenState() error {
	char, err := t.Consume()
	if err != nil {
		return nil
	}

	if unicode.IsLetter(char) {
		err = t.reader.UnreadRune()
		if err != nil {
			return err
		}
		t.workingToken = &Token{token: Token_EndTag, value: ""}
		t.state = State_ScriptData_EndTagName
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.tokens = append(t.tokens, NewCharToken('<'), NewCharToken('/'))
	t.state = State_ScriptData

	return nil
}

// https://html.spec.whatwg.org/#script-data-end-tag-name-state
func (t *Tokenizer) ScriptData_EndTagNameState() error {
	char, err := t.Consume()
	if err != nil {
		return nil
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		if t.HasApproriateEndTagToken() {
			t.state = State_BeforeAttributeName
			return nil
		}
	}

	if char == '/' {
		if t.HasApproriateEndTagToken() {
			t.state = State_SelfClosingStartTag
			return nil
		}
	}

	if char == '>' {
		if t.HasApproriateEndTagToken() {
			t.state = State_Data
			return nil
		}
	}

	if unicode.IsLetter(char) {
		if unicode.IsUpper(char) {
			t.workingToken.value += string(unicode.ToLower(char))
			t.tempbuffer += string(char)
			return nil
		}

		t.workingToken.value += string(char)
		t.tempbuffer += string(char)

		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_ScriptData
	t.tokens = append(t.tokens, NewCharToken('<'), NewCharToken('/'))

	for x := range t.tempbuffer {
		t.tokens = append(t.tokens, NewCharToken(rune(t.tempbuffer[x])))
	}

	return nil
}

// https://html.spec.whatwg.org/#script-data-escape-start-state
func (t *Tokenizer) ScriptData_Escape_StartState() error {
	char, err := t.Consume()
	if err != nil {
		return nil
	}

	if char == '/' {
		t.state = State_ScriptData_EscapeStartDash
		t.tokens = append(t.tokens, NewCharToken('-'))
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_ScriptData

	return nil
}

// https://html.spec.whatwg.org/#script-data-escape-start-dash-state
func (t *Tokenizer) ScriptData_Escape_StartDashState() error {
	char, err := t.Consume()
	if err != nil {
		return nil
	}

	if char == '-' {
		t.state = State_ScriptData_EscapedDashDash
		t.tokens = append(t.tokens, NewCharToken('-'))
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_ScriptData

	return nil
}

// https://html.spec.whatwg.org/#script-data-escaped-state
func (t *Tokenizer) ScriptData_Escaped_State() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return io.EOF
	}

	if char == '-' {
		t.state = State_ScriptData_EscapedDash
		t.tokens = append(t.tokens, NewCharToken('-'))
		return nil
	}

	if char == '\u0000' {
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewCharToken(char))

	return nil
}

// https://html.spec.whatwg.org/#script-data-escaped-dash-state
func (t *Tokenizer) ScriptData_Escaped_Dash_State() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return io.EOF
	}

	if char == '-' {
		t.state = State_ScriptData_EscapedDashDash
		t.tokens = append(t.tokens, NewCharToken('-'))
		return nil
	}

	if char == '<' {
		t.state = State_ScriptData_EscapedLessThanSign
		return nil
	}

	if char == '\u0000' {
		t.state = State_ScriptData_Escaped
		t.tokens = append(t.tokens, NewEOFToken())
		return nil
	}

	t.state = State_ScriptData_Escaped
	t.tokens = append(t.tokens, NewCharToken(char))

	return nil
}

// https://html.spec.whatwg.org/#script-data-escaped-dash-dash-state
func (t *Tokenizer) ScriptData_Escaped_DashDash_State() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return io.EOF
	}

	if char == '-' {
		t.tokens = append(t.tokens, NewCharToken('-'))
		return nil
	}

	if char == '<' {
		t.state = State_ScriptData_EscapedLessThanSign
		return nil
	}

	if char == '>' {
		t.state = State_ScriptData
		t.tokens = append(t.tokens, NewCharToken('>'))
		return nil
	}

	if char == '\u0000' {
		t.state = State_ScriptData_Escaped
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.state = State_ScriptData_Escaped
	t.tokens = append(t.tokens, NewCharToken(char))

	return nil
}

// https://html.spec.whatwg.org/#script-data-escaped-less-than-sign-state
func (t *Tokenizer) ScriptData_Escaped_LessThanSign_State() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if char == '/' {
		t.tempbuffer = ""
		t.state = State_ScriptData_EscapedEndTagOpen
		return nil
	}

	if unicode.IsLetter(char) {
		err = t.reader.UnreadRune()
		if err != nil {
			return err
		}
		t.tempbuffer = ""
		t.tokens = append(t.tokens, NewCharToken('<'))
		t.state = State_ScriptData_DoubleEscapeStart
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.tokens = append(t.tokens, NewCharToken('<'))
	t.state = State_ScriptData_Escaped

	return nil
}

// https://html.spec.whatwg.org/#script-data-escaped-end-tag-open-state
func (t *Tokenizer) ScriptData_Escaped_EndTagOpen_State() error {
	char, err := t.Consume()
	if err != nil {
		return nil
	}

	if unicode.IsLetter(char) {
		err = t.reader.UnreadRune()
		if err != nil {
			return err
		}
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		t.state = State_ScriptData_EscapedEndTagName

		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.tokens = append(t.tokens, NewCharToken('<'), NewCharToken('/'))
	t.state = State_ScriptData_Escaped

	return nil
}

// https://html.spec.whatwg.org/#script-data-escaped-end-tag-name-state
func (t *Tokenizer) ScriptData_Escaped_EndTagName_State() error {
	char, err := t.Consume()
	if err != nil {
		return nil
	}

	if unicode.IsLetter(char) {
		if unicode.IsUpper(char) {
			t.workingToken.value += string(unicode.ToLower(char))
			t.tempbuffer += string(char)
			return nil
		}

		t.workingToken.value += string(char)
		t.tempbuffer += string(char)
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		if t.HasApproriateEndTagToken() {
			t.state = State_BeforeAttributeName
			return nil
		}
	}

	if char == '/' {
		if t.HasApproriateEndTagToken() {
			t.state = State_SelfClosingStartTag
			return nil
		}
	}

	if char == '>' {
		if t.HasApproriateEndTagToken() {
			t.state = State_Data
			return nil
		}
	}

	t.tokens = append(t.tokens, NewCharToken('<'), NewCharToken('/'))

	for x := range t.tempbuffer {
		t.tokens = append(t.tokens, NewCharToken(rune(t.tempbuffer[x])))
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_ScriptData_Escaped

	return nil
}

// https://html.spec.whatwg.org/#script-data-double-escape-start-state
func (t *Tokenizer) ScriptData_Double_Escaped_Start_State() error {
	char, err := t.Consume()
	if err != nil {
		return nil
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' ||
		char == '\u0020' || char == '/' || char == '>' {

		if t.tempbuffer == "script" {
			t.state = State_ScriptData_DoubleEscaped
			return nil
		}

		t.state = State_ScriptData_Escaped
		t.tokens = append(t.tokens, NewCharToken(char))
		return nil
	}

	if unicode.IsLetter(char) {
		if unicode.IsUpper(char) {
			t.tempbuffer += string(unicode.ToLower(char))
		} else {
			t.tempbuffer += string(char)
		}

		t.tokens = append(t.tokens, NewCharToken(char))

		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_ScriptData_Escaped

	return nil
}

// https://html.spec.whatwg.org/#script-data-double-escaped-state
func (t *Tokenizer) ScriptData_Double_Esccaped_State() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return io.EOF
	}

	if char == '-' {
		t.state = State_ScriptData_DoubleEscapedDash
		t.tokens = append(t.tokens, NewCharToken('-'))
		return nil
	}

	if char == '<' {
		t.state = State_ScriptData_DoubleEscapedLessThanSign
		t.tokens = append(t.tokens, NewCharToken('<'))
		return nil
	}

	if char == '\u0000' {
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.tokens = append(t.tokens, NewCharToken(char))

	return nil
}

// https://html.spec.whatwg.org/#script-data-double-escaped-dash-state
func (t *Tokenizer) ScriptData_Double_Escaped_Dash_State() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return io.EOF
	}

	if char == '-' {
		t.state = State_ScriptData_DoubleEscapedDashDash
		t.tokens = append(t.tokens, NewCharToken('-'))
		return nil
	}

	if char == '<' {
		t.state = State_ScriptData_DoubleEscapedLessThanSign
		t.tokens = append(t.tokens, NewCharToken('<'))
		return nil
	}

	if char == '\u0000' {
		t.state = State_ScriptData_Escaped
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.state = State_ScriptData_DoubleEscaped
	t.tokens = append(t.tokens, NewCharToken(char))

	return nil
}

// https://html.spec.whatwg.org/#script-data-double-escaped-dash-dash-state
func (t *Tokenizer) ScriptData_Double_Escaped_DashDash_State() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return io.EOF
	}

	if char == '-' {
		t.tokens = append(t.tokens, NewCharToken('-'))
		return nil
	}

	if char == '<' {
		t.tokens = append(t.tokens, NewCharToken('<'))
		t.state = State_ScriptData_DoubleEscapedLessThanSign
		return nil
	}

	if char == '\u0000' {
		t.state = State_ScriptData_DoubleEscaped
		t.tokens = append(t.tokens, NewReplacementToken())
		return nil
	}

	t.state = State_ScriptData_DoubleEscaped
	t.tokens = append(t.tokens, NewCharToken(char))

	return nil
}

// https://html.spec.whatwg.org/#script-data-double-escaped-less-than-sign-state
func (t *Tokenizer) ScriptData_Double_Escaped_LessThanSign_State() error {
	char, err := t.Consume()
	if err != nil {
		return nil
	}

	if char == '/' {
		t.tempbuffer = ""
		t.state = State_ScriptData_DoubleEscapeEnd
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_ScriptData_DoubleEscaped

	return nil
}

// https://html.spec.whatwg.org/#script-data-double-escape-end-state
func (t *Tokenizer) ScriptData_Double_Escaped_End_State() error {
	char, err := t.Consume()
	if err != nil {
		return nil
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' ||
		char == '\u0020' || char == '/' || char == '>' {
		if t.tempbuffer == "script" {
			t.state = State_ScriptData_Escaped
			return nil
		}

		t.state = State_ScriptData_DoubleEscaped
		t.tokens = append(t.tokens, NewCharToken(char))
		return nil
	}

	if unicode.IsLetter(char) {
		if unicode.IsUpper(char) {
			t.tempbuffer += string(unicode.ToLower(char))
			t.tokens = append(t.tokens, NewCharToken(char))
			return nil
		}

		t.tempbuffer += string(char)
		t.tokens = append(t.tokens, NewCharToken(char))

		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_ScriptData_DoubleEscaped

	return nil
}

// https://html.spec.whatwg.org/#before-attribute-name-state
func (t *Tokenizer) BeforeAttributeNameState() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if char == '\t' || char == '\u000A' ||
		char == '\u000C' || char == '\u0020' {
		return nil
	}

	if char == '/' || char == '>' || isEOF {
		err = t.reader.UnreadRune()
		if err != nil {
			return err
		}
		t.state = State_AfterAttributeName

		return nil
	}

	if char == '=' {
		t.state = State_AttributeName
		t.cAttrName = string(char)
		t.cAttrValue = ""

		return nil
	}

	return nil
}

// https://html.spec.whatwg.org/#attribute-name-state
func (t *Tokenizer) AttributeNameState() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' || char == '/' || char == '>' || isEOF {
		err = t.reader.UnreadRune()
		if err != nil {
			return err
		}
		t.state = State_AfterAttributeName
		return nil
	}

	if char == '=' {
		t.state = State_BeforeAttributeName
		return nil
	}

	if unicode.IsLetter(char) && unicode.IsUpper(char) {
		t.cAttrName += string(unicode.ToLower(char))
		return nil
	}

	if char == '\u0000' {
		t.cAttrName += string(rune('\uFFFD'))
		return nil
	}

	// This is an unexpected-character-in-attribute-name parse error. Treat it as per the "anything else" entry below.
	//if char == '"' || char == '\'' || char == '<' {}

	t.cAttrName += string(char)

	return nil
}

// https://html.spec.whatwg.org/#after-attribute-name-state
func (t *Tokenizer) AfterAttributeNameState() error {
	char, err := t.Consume()

	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	//  eof-in-tag parse error.
	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return err
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		return nil
	}

	if char == '/' {
		t.state = State_SelfClosingStartTag
		return nil
	}

	if char == '=' {
		t.state = State_BeforeAttributeName
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	t.cAttrName = ""
	t.cAttrValue = ""
	t.state = State_AttributeName

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}

	return nil
}

// https://html.spec.whatwg.org/#before-attribute-value-state
func (t *Tokenizer) BeforeAttributeValueState() error {
	char, err := t.Consume()
	if err != nil {
		return err
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		return nil
	}

	if char == '"' {
		t.state = State_AttributValue_DoubleQuoted
		return nil
	}

	if char == '\'' {
		t.state = State_AttributValue_SingleQuoted
		return nil
	}

	if char == '>' {
		t.state = State_Data

		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_AttributValue_Unquoted

	return nil
}

// https://html.spec.whatwg.org/#attribute-value-(double-quoted)-state
func (t *Tokenizer) AttributeValue_DoubleQuote_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return err
	}

	if char == '"' {
		t.state = State_AfterAttributeValue_Quoted
		return nil
	}

	if char == '&' {
		t.rstate = State_AttributValue_DoubleQuoted
		t.state = State_CharacterReference
		return nil
	}

	if char == '\u0000' {
		t.cAttrValue += string(rune('\ufffd'))
		return nil
	}

	t.cAttrValue += string(char)

	return nil
}

// https://html.spec.whatwg.org/#attribute-value-(single-quoted)-state
func (t *Tokenizer) AttributeValue_Signle_Quote_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return err
	}

	if char == '\'' {
		t.state = State_AfterAttributeValue_Quoted
		return nil
	}

	if char == '&' {
		t.rstate = State_AttributValue_SingleQuoted
		t.state = State_CharacterReference
		return nil
	}

	if char == '\u0000' {
		t.cAttrValue += string(rune('\uFFFD'))
		return nil
	}

	t.cAttrValue += string(char)

	return nil
}

// https://html.spec.whatwg.org/#attribute-value-(unquoted)-state
func (t *Tokenizer) AttributeValue_Unquoted_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return err
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		t.state = State_BeforeAttributeName
		return nil
	}

	if char == '&' {
		t.rstate = State_AttributValue_Unquoted
		t.state = State_CharacterReference
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	if char == '\u0000' {
		t.cAttrValue += string(rune('\uFFFD'))
		return nil
	}

	//  unexpected-character-in-unquoted-attribute-value parse error. Treat it as per the "anything else" entry below.
	//if char == '"' || char == '\'' || char == '<' || char == '=' || char == '`' {}

	t.cAttrValue += string(char)

	return nil
}

// https://html.spec.whatwg.org/#after-attribute-value-(quoted)-state
func (t *Tokenizer) AfterAttributeValue_QuotedState() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return err
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		t.state = State_BeforeAttributeName
		return nil
	}

	if char == '/' {
		t.state = State_SelfClosingStartTag
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	//  missing-whitespace-between-attributes parse error.
	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_BeforeAttributeName

	return nil
}

// https://html.spec.whatwg.org/#self-closing-start-tag-state
func (t *Tokenizer) SelfClosingStartTagState() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return err
	}

	if char == '>' {
		t.workingToken.selfClosing = true
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_BeforeAttributeName

	return nil
}

// https://html.spec.whatwg.org/#bogus-comment-state
func (t *Tokenizer) BogusCommentState() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil
		return err
	}

	if char == '>' {
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	if char == '\u0000' {
		t.workingToken.value += string(utf8.RuneError)
		return nil
	}

	t.workingToken.value += string(char)

	return nil
}

// https://html.spec.whatwg.org/#markup-declaration-open-state
func (t *Tokenizer) MarkupDeclarationOpenState() error {

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

		t.state = State_CommentStart
		t.workingToken = &Token{token: Token_Comment, value: ""}

		return nil
	}

	runes, err = t.reader.Peek(7)
	if err != nil {
		return err
	}

	a := strings.ToUpper(string(runes))
	if a == "DOCTYPE" {
		t.state = State_DOCTYPE

		rs := make([]rune, 7)

		_, err := t.reader.Read(rs)

		if err != nil {
			return nil
		}

		return nil
	}

	runes, err = t.reader.Peek(7)
	if err != nil {
		return err
	}

	a = strings.ToUpper(string(runes))
	if a == "[CDATA[" {
		rs := make([]rune, 7)
		_, err := t.reader.Read(rs)
		if err != nil {
			return err
		}

		//TODO: adjeusted current node check && html namespace check
		// t.state = State_CDATA_Section

		t.tokens = append(t.tokens, Token{
			token: Token_Comment,
			value: string(rs),
		})
		t.state = State_BogusComment
		return nil
	}

	t.workingToken = &Token{token: Token_Comment, value: ""}
	t.state = State_BogusComment

	return nil
}

// https://html.spec.whatwg.org/#comment-start-state
func (t *Tokenizer) Comment_Start_State() error {
	char, err := t.Consume()

	if err != nil {
		return err
	}

	if char == '-' {
		t.state = State_CommentStartDash
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_Comment

	return nil
}

// https://html.spec.whatwg.org/#comment-start-dash-state
func (t *Tokenizer) Comment_StartDash_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil
		return nil
	}

	if char == '-' {
		t.state = State_CommentEnd
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	t.workingToken.value += "-"
	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_Comment

	return nil
}

// https://html.spec.whatwg.org/#comment-state
func (t *Tokenizer) Comment_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil
		return err
	}

	if char == '<' {
		t.workingToken.value += string(char)
		t.state = State_CommentLessThanSign
		return nil
	}

	if char == '-' {
		t.state = State_CommentEndDash
		return nil
	}

	if char == '\u0000' {
		t.workingToken.value += string(rune('\uFFFD'))
		return nil
	}

	t.workingToken.value += string(char)

	return nil
}

// https://html.spec.whatwg.org/#comment-less-than-sign-state
func (t *Tokenizer) Comment_LessThanSignState() error {
	char, err := t.Consume()

	if err != nil {
		return err
	}

	if char == '!' {
		t.workingToken.value += "!"
		t.state = State_CommentLessThanSignBang
		return nil
	}

	if char == '<' {
		t.workingToken.value += "<"
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_Comment

	return nil
}

// https://html.spec.whatwg.org/#comment-less-than-sign-bang-state
func (t *Tokenizer) Comment_LessThanSign_Bang_State() error {
	char, err := t.Consume()

	if err != nil {
		return err
	}

	if char == '-' {
		t.state = State_CommentLessThanSignBangDash
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_Comment

	return nil
}

// https://html.spec.whatwg.org/#comment-less-than-sign-bang-dash-state
func (t *Tokenizer) Comment_LessThanSign_BangDash_State() error {
	char, err := t.Consume()

	if err != nil {
		return err
	}

	if char == '-' {
		t.state = State_CommentLessThanSignBangDashDash
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_CommentEndDash

	return nil
}

// https://html.spec.whatwg.org/#comment-less-than-sign-bang-dash-dash-state
func (t *Tokenizer) Comment_LessThanSign_BangDashDash_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF || char == '>' {
		err = t.reader.UnreadRune()
		if err != nil {
			return err
		}
		t.state = State_CommentEnd
		return nil
	}

	// This is a nested-comment parse error.
	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_CommentEnd

	return nil
}

// https://html.spec.whatwg.org/#comment-end-dash-state
func (t *Tokenizer) Comment_EndDash_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil
		return err
	}

	if char == '-' {
		t.state = State_CommentEnd
		return nil
	}

	t.workingToken.value += "-"
	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_Comment

	return nil
}

// https://html.spec.whatwg.org/#comment-end-state
func (t *Tokenizer) Comment_End_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		return err
	}

	if char == '>' {
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	if char == '!' {
		t.state = State_CommentEndBang
		return nil
	}

	if char == '-' {
		t.workingToken.value += "-"
		return nil
	}

	t.workingToken.value += "--"
	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_Comment

	return nil
}

// https://html.spec.whatwg.org/#comment-end-bang-state
func (t *Tokenizer) Comment_EndBang_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if char == '-' {
		t.workingToken.value += "--!"
		t.state = State_CommentEnd
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	t.workingToken.value += "--!"
	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_Comment

	return nil
}

// https://html.spec.whatwg.org/#doctype-state
func (t *Tokenizer) DOCTYPE_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, Token{token: Token_DOCTYPE}, NewEOFToken())
		return nil
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		t.state = State_BeforeDOCTYPEName
		return nil
	}

	if char == '>' {
		t.state = State_BeforeDOCTYPEName
		err = t.reader.UnreadRune()
		return err
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}
	t.state = State_BeforeDOCTYPEName

	return nil
}

// https://html.spec.whatwg.org/#before-doctype-name-state
func (t *Tokenizer) BeforeDocTypeNameState() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, Token{token: Token_DOCTYPE})
		t.state = State_Data
		return nil
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		return nil
	}

	if unicode.IsLetter(char) && unicode.IsUpper(char) {
		t.workingToken = &Token{token: Token_DOCTYPE, value: string(unicode.ToLower(char))}
		t.state = State_DOCTYPE_Name
		return nil
	}

	if char == '\u0000' {
		t.workingToken = &Token{token: Token_DOCTYPE, value: string(rune('\uFFFD'))}
		t.state = State_DOCTYPE_Name
		return nil
	}

	if char == '>' {
		t.tokens = append(t.tokens, Token{token: Token_DOCTYPE})
		t.state = State_Data
		return nil
	}

	t.workingToken = &Token{token: Token_DOCTYPE, value: string(char)}
	t.state = State_DOCTYPE_Name
	return nil
}

// https://html.spec.whatwg.org/#doctype-name-state
func (t *Tokenizer) DOCTYPE_NameState() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil

		return err
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		t.state = State_AfterDOCTYPE_Name
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil

		return nil
	}

	if unicode.IsLetter(char) && unicode.IsUpper(char) {
		t.workingToken.value += string(unicode.ToLower(char))
		return nil
	}

	if char == '\u0000' {
		t.workingToken.value += string(rune('\uFFFD'))
		return nil
	}

	t.workingToken.value += string(char)

	return nil
}

// https://html.spec.whatwg.org/#after-doctype-name-state
func (t *Tokenizer) After_DOCTYPE_Name() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil

		return err
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil

		return nil
	}

	ptest, err := t.reader.Peek(6)
	if err != nil {
		return err
	}

	text := strings.ToUpper(string(ptest))

	if text == "PUBLIC" {
		v := make([]rune, 6)

		_, err = t.reader.Read(v)
		if err != nil {
			return err
		}

		t.state = State_AfterDOCTYPE_PublicKeyword
		return nil
	}

	if text == "SYSTEM" {
		v := make([]rune, 6)

		_, err = t.reader.Read(v)
		if err != nil {
			return err
		}

		t.state = State_AfterDOCTYPE_SystemKeyword
		return nil
	}

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}

	t.state = State_BogusDOCTYPE

	return nil
}

// https://html.spec.whatwg.org/#after-doctype-public-keyword-state
func (t *Tokenizer) AfterDOCTYPE_PublicKeywordState() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		t.state = State_BeforeDOCTYPE_PublicIDentifier
		return nil
	}

	if char == '"' {
		t.workingToken.state["publicIdentifier"] = ""
		t.state = State_DOCTYPE_PublicIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		t.workingToken.state["publicIdentifier"] = ""
		t.state = State_DOCTYPE_PublicIdentifier_SingleQuoted
		return nil
	}

	if char == '>' {
		t.workingToken.state["forceQuirks"] = "true"
		t.state = State_Data

		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	t.workingToken.state["forceQuirks"] = "true"
	t.state = State_BogusDOCTYPE

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}

	return nil
}

// https://html.spec.whatwg.org/#before-doctype-public-identifier-state
func (t *Tokenizer) Before_DOCTYPE_PublicIdentifierState() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.workingToken.state["forceQuirks"] = "true"
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil
		t.state = State_Data
		return nil
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		return nil
	}

	if char == '"' {
		t.workingToken.state["publicIdentifier"] = ""
		t.state = State_DOCTYPE_PublicIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		t.workingToken.state["publicIdentifier"] = ""
		t.state = State_DOCTYPE_PublicIdentifier_SingleQuoted
		return nil
	}

	if char == '>' {
		t.workingToken.state["forceQuirks"] = "true"
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	t.workingToken.state["forceQuirks"] = "true"
	t.state = State_BogusDOCTYPE

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}

	return nil
}

// https://html.spec.whatwg.org/#doctype-public-identifier-(double-quoted)-state
func (t *Tokenizer) DOCTYPE_PublicIdentifier_DoubleQuoted_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.workingToken.state["forceQuirks"] = "true"
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil
		return io.EOF
	}

	if char == '"' {
		t.state = State_AfterDOCTYPE_PublicIdentifier
		return nil
	}

	if char == '\u0000' {
		t.workingToken.state["publicIdentifier"] += string(rune('\uFFFD'))
		return nil
	}

	if char == '>' {
		t.workingToken.state["forceQuirks"] = "true"
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		t.state = State_Data
		return nil
	}

	t.workingToken.state["publicIdentifier"] += string(char)

	return nil
}

// https://html.spec.whatwg.org/#doctype-public-identifier-(single-quoted)-state
func (t *Tokenizer) DOCKTYPE_PublicIdentifier_SingleQuoted_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if char == '\'' {
		t.state = State_AfterDOCTYPE_PublicIdentifier
		return nil
	}

	if char == '\u0000' {
		t.workingToken.state["publicIdentifier"] += string(char)
		return nil
	}

	if char == '>' {
		t.workingToken.state["forceQuirks"] = "true"
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	t.workingToken.state["publicIdentifier"] += string(char)

	return nil
}

// https://html.spec.whatwg.org/#after-doctype-public-identifier-state
func (t *Tokenizer) After_DOCTYPE_PublicIdentifier_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.workingToken.state["forceQuirks"] = "true"
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil
		return io.EOF
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		t.state = State_BetweenDOCTYPE_PublicAndSystemIdentifiers
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	if char == '"' {
		t.workingToken.state["publicIdentifier"] = ""
		t.state = State_DOCTYPE_SystemIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		t.workingToken.state["publicIdentifier"] = ""
		t.state = State_DOCTYPE_SystemIdentifier_SingleQuoted
		return nil
	}

	t.workingToken.state["forceQuirks"] = "true"
	t.state = State_BogusDOCTYPE

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}

	return nil
}

// https://html.spec.whatwg.org/#between-doctype-public-and-system-identifiers-state
func (t *Tokenizer) Between_DOCTYPE_PublicAndSystemIdent_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.workingToken.state["forceQuirks"] = "true"
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil

		return io.EOF
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		return nil
	}

	if char == '>' {
		t.state = State_Data

		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil

		return nil
	}

	if char == '"' {
		t.workingToken.state["systemIdentifier"] = ""
		t.state = State_DOCTYPE_SystemIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		t.workingToken.state["systemIdentifier"] = ""
		t.state = State_DOCTYPE_SystemIdentifier_SingleQuoted
		return nil
	}

	t.workingToken.state["forceQuirks"] = "true"
	t.state = State_BogusDOCTYPE

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}

	return nil
}

// https://html.spec.whatwg.org/#after-doctype-system-keyword-state
func (t *Tokenizer) After_DOCTYPE_SystemKeyword_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.workingToken.state["forceQuirks"] = "true"
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil
		return nil
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		return nil
	}

	if char == '"' {
		t.workingToken.state["systemIdentifier"] = ""
		t.state = State_DOCTYPE_SystemIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		t.workingToken.state["systemIdentifier"] = ""
		t.state = State_DOCTYPE_SystemIdentifier_SingleQuoted
		return nil
	}

	if char == '>' {
		t.workingToken.state["forceQuirks"] = "true"
		t.state = State_Data

		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	t.workingToken.state["forceQuirks"] = "true"
	t.state = State_BogusDOCTYPE

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}

	return nil
}

// https://html.spec.whatwg.org/#before-doctype-system-identifier-state
func (t *Tokenizer) Before_DOCTYPE_SystemIdentifier_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.workingToken.state["forceQuirks"] = "true"
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil
		return nil
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		return nil
	}

	if char == '"' {
		t.workingToken.state["systemIdentifier"] = ""
		t.state = State_DOCTYPE_SystemIdentifier_DoubleQuoted
		return nil
	}

	if char == '\'' {
		t.workingToken.state["systemIdentifier"] = ""
		t.state = State_DOCTYPE_SystemIdentifier_SingleQuoted
		return nil
	}

	if char == '>' {
		t.workingToken.state["forceQuirks"] = "true"
		t.state = State_Data

		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	t.workingToken.state["forceQuirks"] = "true"
	t.state = State_BogusDOCTYPE

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}

	return nil
}

// https://html.spec.whatwg.org/#doctype-system-identifier-(double-quoted)-state
func (t *Tokenizer) DOCTYPE_SystemIdentifier_DoubleQuoted_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.workingToken.state["forceQuirks"] = "true"
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil
		return nil
	}

	if char == '"' {
		t.state = State_AfterDOCTYPE_SystemIdentifier
		return nil
	}

	if char == '\u0000' {
		t.workingToken.state["systemIdentifier"] += string(rune('\uFFFD'))
		return nil
	}

	if char == '>' {
		t.workingToken.state["forceQuirks"] = "true"
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	t.workingToken.state["systemIdentifier"] += string(char)

	return nil
}

// https://html.spec.whatwg.org/#doctype-system-identifier-(single-quoted)-state
func (t *Tokenizer) DOCTYPE_SystemIdentifier_SingleQuoted_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.workingToken.state["forceQuirks"] = "true"
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil
		return nil
	}

	if char == '\'' {
		t.state = State_AfterDOCTYPE_SystemIdentifier
		return nil
	}

	if char == '\u0000' {
		t.workingToken.state["systemIdentifier"] += string(rune('\uFFFD'))
		return nil
	}

	if char == '>' {
		t.workingToken.state["forceQuirks"] = "true"
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	t.workingToken.state["systemIdentifier"] += string(char)

	return nil
}

// https://html.spec.whatwg.org/#after-doctype-system-identifier-state
func (t *Tokenizer) After_DOCTYPE_SystemIdentifer_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.workingToken.state["forceQuirks"] = "true"
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil
		return nil
	}

	if char == '\t' || char == '\u000A' || char == '\u000C' || char == '\u0020' {
		return nil
	}

	if char == '>' {
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	t.state = State_BogusDOCTYPE

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}

	return nil
}

// https://html.spec.whatwg.org/#bogus-doctype-state
func (t *Tokenizer) Bogus_DOCTYPE_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, *t.workingToken, NewEOFToken())
		t.workingToken = nil

		return io.EOF
	}

	if char == '>' {
		t.state = State_Data
		t.tokens = append(t.tokens, *t.workingToken)
		t.workingToken = nil
		return nil
	}

	return nil
}

// https://html.spec.whatwg.org/#cdata-section-state
func (t *Tokenizer) CDATA_Section_State() error {
	char, err := t.Consume()
	isEOF := err == io.EOF
	if err != nil && !isEOF {
		return err
	}

	if isEOF {
		t.tokens = append(t.tokens, NewEOFToken())
		return io.EOF
	}

	if char == ']' {
		t.state = State_CDATA_SectionBracket
		return nil
	}

	t.tokens = append(t.tokens, NewCharToken(char))

	return nil
}

// https://html.spec.whatwg.org/#cdata-section-bracket-state
func (t *Tokenizer) CDATA_Section_Bracket_State() error {
	char, err := t.Consume()
	if err != nil {
		return err
	}

	if char == ']' {
		t.state = State_CDATA_SectionEnd
		return nil
	}

	t.tokens = append(t.tokens, NewCharToken(']'))
	t.state = State_CDATA_Section

	err = t.reader.UnreadRune()
	if err != nil {
		return nil
	}

	return nil
}

// https://html.spec.whatwg.org/#cdata-section-end-state
func (t *Tokenizer) CDATA_Section_End_State() error {
	char, err := t.Consume()
	if err != nil {
		return err
	}

	if char == ']' {
		t.tokens = append(t.tokens, NewCharToken(']'))
		return nil
	}

	if char == '>' {
		t.state = State_Data
		return nil
	}

	t.tokens = append(t.tokens, NewCharToken(']'), NewCharToken(']'))
	t.state = State_CDATA_Section

	err = t.reader.UnreadRune()
	if err != nil {
		return err
	}

	return nil
}

// https://html.spec.whatwg.org/#character-reference-state
func (t *Tokenizer) CharacterReferenceState() error {
	t.tempbuffer = "&"

	char, err := t.Consume()
	if err != nil {
		return err
	}

	if unicode.IsNumber(char) || unicode.IsLetter(char) {
		t.state = State_NamedCharacterReference

		err = t.reader.UnreadRune()
		return err
	}

	if char == '#' {
		t.tempbuffer += "#"
		t.state = State_NumericCharacterReference
		return nil
	}

	if t.cAttrName != "" {
		t.cAttrName += t.tempbuffer
	} else {
		for x := range t.tempbuffer {
			t.tokens = append(t.tokens, NewCharToken(rune(t.tempbuffer[x])))
		}
	}

	t.state = t.rstate

	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#named-character-reference-state
func (t *Tokenizer) NamedCharacterReferenceState() error {

	// TODO: impl parsing

	t.state = State_AmbiguousAmpersand

	// flush

	return nil
}

// https://html.spec.whatwg.org/#ambiguous-ampersand-state
func (t *Tokenizer) NumericCharacterReferenceState() error {
	char, err := t.Consume()
	if err != nil {
		return err
	}

	if unicode.IsLetter(char) || unicode.IsNumber(char) {
		if t.cAttrValue != "" {
			t.cAttrValue += string(char)
			return nil
		}

		t.tokens = append(t.tokens, NewCharToken(char))
		return nil
	}

	//  unknown-named-character-reference parse error.
	/*if char == ';' {
		t.state = t.rstate
		return t.reader.UnreadRune()
	}*/

	t.state = t.rstate
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#numeric-character-reference-state
func (t *Tokenizer) HexadecimalCharacterReferenceStartState() error {
	char, err := t.Consume()
	if err != nil {
		return err
	}

	if char == '\u0078' || char == '\u0058' {
		t.tempbuffer += string(char)
		t.state = State_HexadeciamCharacterReferenceStart
		return nil
	}

	t.state = State_DecimalCharacterReferenceStart
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#hexadecimal-character-reference-start-state
func (t *Tokenizer) DecimalCharacterReferenceStartState() error {
	_, err := t.Consume()
	if err != nil {
		return err
	}

	//TODO if hex digit
	// reconusme
	// state = hexadeciam

	// flush
	t.state = t.rstate
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#hexadecimal-character-reference-state
func (t *Tokenizer) HexadecimalCharacterReferenceState() error {
	char, err := t.Consume()
	if err != nil {
		return err
	}

	if char == ';' {
		t.state = State_NumericCharacterReferenceEnd
		return nil
	}

	t.state = State_NumericCharacterReferenceEnd
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#decimal-character-reference-state
func (t *Tokenizer) DeciamalCharacterReferenceState() error {
	char, err := t.Consume()
	if err != nil {
		return err
	}

	if char == ';' {
		t.state = State_NumericCharacterReferenceEnd
		return nil
	}

	t.state = State_NumericCharacterReferenceEnd
	return t.reader.UnreadRune()
}

// https://html.spec.whatwg.org/#numeric-character-reference-end-state
func (t *Tokenizer) NumericCharacterReferenceEndState() error {
	t.state = t.rstate
	return nil
}
