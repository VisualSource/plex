package html

import (
	"slices"

	"github.com/VisualSource/plex/internal/html_tokenizer"
)

func isAny[T comparable](value T, values ...T) bool {
	return slices.Contains(values, value)
}

func isAnyRune(token html_tokenizer.Token, chars ...rune) bool {
	tag, ok := token.(*html_tokenizer.TokenCharacter)
	if !ok {
		return false
	}

	return slices.Contains(chars, tag.Data)
}

func isAnyStartTag(token html_tokenizer.Token, tags ...string) bool {
	tag, ok := token.(*html_tokenizer.TagToken)

	if !ok || tag.IsEndTag() {
		return false
	}

	return slices.Contains(tags, tag.Name)
}

func isAnyTag(token *html_tokenizer.TagToken, tags ...string) bool {
	return slices.Contains(tags, token.Name)
}

func isAnyEndTag(token html_tokenizer.Token, tags ...string) bool {
	tag, ok := token.(*html_tokenizer.TagToken)

	if !ok || tag.IsStartTag() {
		return false
	}

	return slices.Contains(tags, tag.Name)
}
