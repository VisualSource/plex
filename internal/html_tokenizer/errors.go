package html_tokenizer

import (
	"fmt"
)

type TokenizerErrorReason string

var (
	ErrEofInTag                                    TokenizerErrorReason = "eof-in-tag"
	ErrDuplicateAttribute                          TokenizerErrorReason = "duplicate-attribute"
	ErrMissingWhitespaceBetweenAttributes          TokenizerErrorReason = "missing-whitespace-between-attributes"
	ErrUnexpectedNullCharacter                     TokenizerErrorReason = "unexpected-null-character"
	ErrEofBeforeTagName                            TokenizerErrorReason = "eof-before-tag-name"
	ErrUnexpectedQuestionMarkInsteadOfTagName      TokenizerErrorReason = "unexpected-question-mark-instead-of-tag-name"
	ErrInvalidFirstCharacterOfTagName              TokenizerErrorReason = "invalid-first-character-of-tag-name"
	ErrMissingEndTagName                           TokenizerErrorReason = "missing-end-tag-name"
	ErrUnexpectedCharacterInAttributeName          TokenizerErrorReason = "unexpected-character-in-attribute-name"
	ErrUnexpectedEqualsSignBeforeAttributeName     TokenizerErrorReason = "unexpected-equals-sign-before-attribute-name"
	ErrMissingAttributeValue                       TokenizerErrorReason = "missing-attribute-value"
	ErrUnexpectedCharacterInUnquotedAttributeValue TokenizerErrorReason = "unexpected-character-in-unquoted-attribute-value"
	ErrCharacterReferenceOutsideUnicodeRange       TokenizerErrorReason = "character-reference-outside-unicode-range"
	ErrSurrogateCharacterReference                 TokenizerErrorReason = "surrogate-character-reference"
	ErrNoncharacterCharacterReference              TokenizerErrorReason = "noncharacter-character-reference"
	ErrControlCharacterReference                   TokenizerErrorReason = "control-character-reference"
	ErrUnknownNamedCharacterReference              TokenizerErrorReason = "unknown-named-character-reference"
	ErrAbsenceOfDigitsInNumericCharacterReference  TokenizerErrorReason = "absence-of-digits-in-numeric-character-reference"
	ErrMissingSemicolonAfterCharacterReference     TokenizerErrorReason = "missing-semicolon-after-character-reference"
	ErrAbruptClosingOfEmptyComment                 TokenizerErrorReason = "abrupt-closing-of-empty-comment"
	ErrEofInComment                                TokenizerErrorReason = "eof-in-comment"
	ErrNestedComment                               TokenizerErrorReason = "nested-comment"
	ErrIncorrectlyClosedComment                    TokenizerErrorReason = "incorrectly-closed-comment"
	ErrIncorrectlyOpenedComment                    TokenizerErrorReason = "incorrectly-opened-comment"
	ErrCdataInHtmlContent                          TokenizerErrorReason = "cdata-in-html-content"
	ErrEofInScriptHtmlCommentLikeText              TokenizerErrorReason = "eof-in-script-html-comment-like-text"
	ErrEofInDoctype                                TokenizerErrorReason = "eof-in-doctype"
	ErrMissingWhitespaceBeforeDoctypeName          TokenizerErrorReason = "missing-whitespace-before-doctype-name"
	ErrMissingDoctypeName                          TokenizerErrorReason = "missing-doctype-name"
	ErrInvalidCharacterSequenceAfterDoctypeName    TokenizerErrorReason = "invalid-character-sequence-after-doctype-name"
	ErrMissingWhitespaceAfterDoctypePublicKeyword  TokenizerErrorReason = "missing-whitespace-after-doctype-public-keyword"
	ErrMissingDoctypePublicIdentifier              TokenizerErrorReason = "missing-doctype-public-identifier"
	ErrMissingQuoteBeforeDoctypePublicIdentifer    TokenizerErrorReason = "missing-quote-before-doctype-public-identifier"
	ErrAbruptDoctypePublicIdentifer                TokenizerErrorReason = "abrupt-doctype-public-identifier"
	ErrAbruptDoctypeSystemIdentifier               TokenizerErrorReason = "abrupt-doctype-system-identifier"
	ErrEndTagWithAttributes                        TokenizerErrorReason = "end-tag-with-attributes"
	ErrUnexpectedSolidusInTag                      TokenizerErrorReason = "unexpected-solidus-in-tag"
	ErrNullCharacterReference                      TokenizerErrorReason = "null-character-reference"
	ErrEofInCDATA                                  TokenizerErrorReason = "eof-in-cdata"
	ErrMissingQuoteBeforeDoctypeSystemIdentifier   TokenizerErrorReason = "missing-quote-before-doctype-system-identifier"
	ErrMissingDoctypeSystemIdentifier              TokenizerErrorReason = "missing-doctype-system-identifier"
	ErrNoWSBetweenDoctypePublicAndSystenIdentifier TokenizerErrorReason = "missing-whitespace-between-doctype-public-and-system-identifiers"
	ErrUnexpectedCharAfterDoctypeSystemIdentifier  TokenizerErrorReason = "unexpected-character-after-doctype-system-identifier"
	ErrMissingWhitespaceAfterDoctypeSystemKeyword  TokenizerErrorReason = "missing-whitespace-after-doctype-system-keyword"
	ErrControlCharacterInInputStream               TokenizerErrorReason = "control-character-in-input-stream"
	ErrEndTagWithTrailingSolidus                   TokenizerErrorReason = "end-tag-with-trailing-solidus"
	ErrNoncharacterInInputStream                   TokenizerErrorReason = "noncharacter-in-input-stream"
	ErrSurrogateInInputStream                      TokenizerErrorReason = "surrogate-in-input-stream"
)

type TokenizerError struct {
	Reason TokenizerErrorReason
	Line   int
	Col    int
}

func (e *TokenizerError) Error() string {
	return fmt.Sprintf("tokenizer error '%s' on line %d col %d", e.Reason, e.Line, e.Col)
}

func NewTokenizerError(reason TokenizerErrorReason, line int, col int) TokenizerError {
	return TokenizerError{
		Reason: reason,
		Line:   line,
		Col:    col,
	}
}
