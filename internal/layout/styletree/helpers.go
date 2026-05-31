package styletree

import (
	css_tokenizer "github.com/VisualSource/plex/internal/css/tokenizer"
	"github.com/Zyko0/go-sdl3/sdl"
)

func parseColor([]css_tokenizer.Token) sdl.FColor {

	return sdl.FColor{}
}

func inheritProperties(propertyMap PropertyMap, parentPropertyMap PropertyMap) PropertyMap {

	for key, value := range parentPropertyMap {
		switch key {
		case "color", "font-family", "font-size", "font-style", "font-variant", "font-weight", "font-size-adjust", "font-stretch", "letter-spacing", "line-height",
			"text-align", "text-indent", "text-shadow", "text-transform", "white-space", "word-break", "word-spacing", "overflow-wrap", "word-wrap", "direction", "unicode-bidi",
			"list-style-image", "list-style-position", "list-style-type", "border-collapse", "border-spacing", "caption-side", "empty-cells", "cursor", "visibility", "quotes", "orphans",
			"widows", "page-break-inside":

			prop, hasProp := propertyMap[key]
			if !hasProp {
				propertyMap[key] = value
			} else if v, ok := prop.Value.(string); ok && v == "inherit" {
				propertyMap[key] = value
			}
		}
	}

	return propertyMap
}
