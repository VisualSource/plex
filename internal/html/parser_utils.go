package html

import (
	"slices"

	tokenizer "github.com/VisualSource/plex/internal/html/tokenizer"
)

func isAny[T comparable](value T, values ...T) bool {
	return slices.Contains(values, value)
}

func isAnyRune(token tokenizer.Token, chars ...rune) bool {
	tag, ok := token.(*tokenizer.TokenCharacter)
	if !ok {
		return false
	}

	return slices.Contains(chars, tag.Data)
}

func isAnyStartTag(token tokenizer.Token, tags ...string) bool {
	tag, ok := token.(*tokenizer.TagToken)

	if !ok || tag.IsEndTag() {
		return false
	}

	return slices.Contains(tags, tag.Name)
}

func isAnyTag(token *tokenizer.TagToken, tags ...string) bool {
	return slices.Contains(tags, token.Name)
}

func isAnyEndTag(token tokenizer.Token, tags ...string) bool {
	tag, ok := token.(*tokenizer.TagToken)

	if !ok || tag.IsStartTag() {
		return false
	}

	return slices.Contains(tags, tag.Name)
}
