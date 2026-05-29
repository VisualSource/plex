package styletree

import (
	"github.com/VisualSource/plex/internal/css/tokenizer"
	"github.com/Zyko0/go-sdl3/sdl"
)

func parseColor([]tokenizer.Token) sdl.FColor {

	return sdl.FColor{}
}

func inheritProperties(propertyMap PropertyMap, parentPropertyMap PropertyMap) PropertyMap {
	/*
			color
			font-family
			font-size
			font-style
			font-variant
			font-weight
			font-size-adjust
			font-stretch
			font (shorthand)
			letter-spacing
			line-height
			text-align
			text-indent
			text-shadow
			text-transform
			white-space
			word-break
			word-spacing
			overflow-wrap / word-wrap
			direction
			unicode-bidi
			list-style-image
			list-style-position
			list-style-type
			list-style (shorthand)
			border-collapse
			border-spacing
			caption-side
			empty-cells
			cursor
		visibility
		quotes
		orphans
		widows
		page-break-inside
	*/

	return propertyMap
}
