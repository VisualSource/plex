package html_tokenizer

import (
	"math"
	"slices"

	"github.com/VisualSource/plex/internal/utils"
)

var NONCHARACTER_CODEPOINTS = []int{
	0xFFFE, 0xFFFF, 0x1FFFE, 0x1FFFF, 0x2FFFE,
	0x2FFFF, 0x3FFFE, 0x3FFFF, 0x4FFFE, 0x4FFFF,
	0x5FFFE, 0x5FFFF, 0x6FFFE, 0x6FFFF, 0x7FFFE,
	0x7FFFF, 0x8FFFE, 0x8FFFF, 0x9FFFE, 0x9FFFF,
	0xAFFFE, 0xAFFFF, 0xBFFFE, 0xBFFFF, 0xCFFFE,
	0xCFFFF, 0xDFFFE, 0xDFFFF, 0xEFFFE, 0xEFFFF,
	0xFFFFE, 0xFFFFF, 0x10FFFE, 0x10FFFF,
}

func isNonCharacterCodepoint(v int) bool {
	return v >= 0xFDD0 && v <= 0xFDEF || slices.Contains(NONCHARACTER_CODEPOINTS, v)
}

func isSurrogate(v int) bool {
	r := (v >= 0xD800 && v <= 0xDBFF) ||
		(v >= 0xDC00 && v <= 0xDFFF)

	return r
}

func isWhitespace(r rune) bool {
	switch r {
	case '\t', '\n', '\f', ' ':
		return true
	}
	return false
}

func wasConsumedAsPartOfAttribute(state TokenizerState) bool {
	switch state {
	case state_AttributValue_DoubleQuoted, state_AttributValue_SingleQuoted, state_AttributValue_Unquoted:
		return true
	default:
		return false
	}
}

// do to go's int (int32?) overflowing we need to do check if we are going to
// overflow so that in the character reference end state we hit the right
// conditions. max character ref is 0x10FFFF so a int should be more
// then enough for are needs
func addNumericDigit(char int, power int, refCode int) int {
	mr, ok := utils.Mul(refCode, power)
	if !ok {
		// if mul overflows then there is not need to try add opt
		// make sure that we will hit invalid ref checks
		return math.MaxInt
	}

	ar, ok := utils.Add(mr, char)
	if !ok {
		return math.MaxInt
	}

	return ar
}
