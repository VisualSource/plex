package cssom

import (
	"strconv"

	css_parser "github.com/VisualSource/plex/internal/css/parser"
	css_tokenizer "github.com/VisualSource/plex/internal/css/tokenizer"
	"github.com/Zyko0/go-sdl3/sdl"
)

// https://drafts.csswg.org/css-color/#color-syntax
func parseColor(tokens []css_tokenizer.Token) (sdl.Color, error) {
	t := nonWS(tokens)

	switch t[0].IsToken() {
	case css_tokenizer.TokenId_Ident:
		value := css_parser.GetTokenValueAsString(t[0])
		switch value {
		case "transparent":
			return sdl.Color{A: 0}, nil
		default:
			co, err := parseNamedColor(t[0])
			if err == nil {
				return co, nil
			}

			return sdl.Color{}, ErrInvalidCssValue
		}
	case css_parser.TokenId_Function:
		return sdl.Color{}, ErrInvalidCssValue
	case css_tokenizer.TokenId_Hash:
		return parseHexColor(t[0])
	default:
		return sdl.Color{}, ErrInvalidCssValue
	}

}

func parseNamedColor(token css_tokenizer.Token) (sdl.Color, error) {
	value := css_parser.GetTokenValueAsString(token)

	switch value {
	case "aliceblue":
		return sdl.Color{R: 240, G: 248, B: 255, A: 255}, nil
	case "antiquewhite":
		return sdl.Color{R: 250, G: 235, B: 215, A: 255}, nil
	case "aqua":
		return sdl.Color{R: 0, G: 255, B: 255, A: 255}, nil
	case "aquamarine":
		return sdl.Color{R: 127, G: 255, B: 212, A: 255}, nil
	case "azure":
		return sdl.Color{R: 240, G: 255, B: 255, A: 255}, nil
	case "beige":
		return sdl.Color{R: 245, G: 245, B: 220, A: 255}, nil
	case "bisque":
		return sdl.Color{R: 255, G: 228, B: 196, A: 255}, nil
	case "black":
		return sdl.Color{R: 0, G: 0, B: 0, A: 255}, nil
	case "blanchedalmond":
		return sdl.Color{R: 255, G: 235, B: 205, A: 255}, nil
	case "blue":
		return sdl.Color{R: 0, G: 0, B: 255, A: 255}, nil
	case "blueviolet":
		return sdl.Color{R: 138, G: 43, B: 226, A: 255}, nil
	case "brown":
		return sdl.Color{R: 165, G: 42, B: 42, A: 255}, nil
	case "burlywood":
		return sdl.Color{R: 222, G: 184, B: 135, A: 255}, nil
	case "cadetblue":
		return sdl.Color{R: 95, G: 158, B: 160, A: 255}, nil
	case "chartreuse":
		return sdl.Color{R: 127, G: 255, B: 0, A: 255}, nil
	case "chocolate":
		return sdl.Color{R: 210, G: 105, B: 30, A: 255}, nil
	case "coral":
		return sdl.Color{R: 255, G: 127, B: 80, A: 255}, nil
	case "cornflowerblue":
		return sdl.Color{R: 100, G: 149, B: 237, A: 255}, nil
	case "cornsilk":
		return sdl.Color{R: 255, G: 248, B: 220, A: 255}, nil
	case "crimson":
		return sdl.Color{R: 220, G: 20, B: 60, A: 255}, nil
	case "cyan":
		return sdl.Color{R: 0, G: 255, B: 255, A: 255}, nil
	case "darkblue":
		return sdl.Color{R: 0, G: 0, B: 139, A: 255}, nil
	case "darkcyan":
		return sdl.Color{R: 0, G: 139, B: 139, A: 255}, nil
	case "darkgoldenrod":
		return sdl.Color{R: 184, G: 134, B: 11, A: 255}, nil
	case "darkgray":
		return sdl.Color{R: 169, G: 169, B: 169, A: 255}, nil
	case "darkgreen":
		return sdl.Color{R: 0, G: 100, B: 0, A: 255}, nil
	case "darkgrey":
		return sdl.Color{R: 169, G: 169, B: 169, A: 255}, nil
	case "darkkhaki":
		return sdl.Color{R: 189, G: 183, B: 107, A: 255}, nil
	case "darkmagenta":
		return sdl.Color{R: 139, G: 0, B: 139, A: 255}, nil
	case "darkolivegreen":
		return sdl.Color{R: 85, G: 107, B: 47, A: 255}, nil
	case "darkorange":
		return sdl.Color{R: 255, G: 140, B: 0, A: 255}, nil
	case "darkorchid":
		return sdl.Color{R: 153, G: 50, B: 204, A: 255}, nil
	case "darkred":
		return sdl.Color{R: 139, G: 0, B: 0, A: 255}, nil
	case "darksalmon":
		return sdl.Color{R: 233, G: 150, B: 122, A: 255}, nil
	case "darkseagreen":
		return sdl.Color{R: 143, G: 188, B: 143, A: 255}, nil
	case "darkslateblue":
		return sdl.Color{R: 72, G: 61, B: 139, A: 255}, nil
	case "darkslategray":
		return sdl.Color{R: 47, G: 79, B: 79, A: 255}, nil
	case "darkslategrey":
		return sdl.Color{R: 47, G: 79, B: 79, A: 255}, nil
	case "darkturquoise":
		return sdl.Color{R: 0, G: 206, B: 209, A: 255}, nil
	case "darkviolet":
		return sdl.Color{R: 148, G: 0, B: 211, A: 255}, nil
	case "deeppink":
		return sdl.Color{R: 255, G: 20, B: 147, A: 255}, nil
	case "deepskyblue":
		return sdl.Color{R: 0, G: 191, B: 255, A: 255}, nil
	case "dimgray":
		return sdl.Color{R: 105, G: 105, B: 105, A: 255}, nil
	case "dimgrey":
		return sdl.Color{R: 105, G: 105, B: 105, A: 255}, nil
	case "dodgerblue":
		return sdl.Color{R: 30, G: 144, B: 255, A: 255}, nil
	case "firebrick":
		return sdl.Color{R: 178, G: 34, B: 34, A: 255}, nil
	case "floralwhite":
		return sdl.Color{R: 255, G: 250, B: 240, A: 255}, nil
	case "forestgreen":
		return sdl.Color{R: 34, G: 139, B: 34, A: 255}, nil
	case "fuchsia":
		return sdl.Color{R: 255, G: 0, B: 255, A: 255}, nil
	case "gainsboro":
		return sdl.Color{R: 220, G: 220, B: 220, A: 255}, nil
	case "ghostwhite":
		return sdl.Color{R: 248, G: 248, B: 255, A: 255}, nil
	case "gold":
		return sdl.Color{R: 255, G: 215, B: 0, A: 255}, nil
	case "goldenrod":
		return sdl.Color{R: 218, G: 165, B: 32, A: 255}, nil
	case "gray":
		return sdl.Color{R: 128, G: 128, B: 128, A: 255}, nil
	case "green":
		return sdl.Color{R: 0, G: 128, B: 0, A: 255}, nil
	case "greenyellow":
		return sdl.Color{R: 173, G: 255, B: 47, A: 255}, nil
	case "grey":
		return sdl.Color{R: 128, G: 128, B: 128, A: 255}, nil
	case "honeydew":
		return sdl.Color{R: 240, G: 255, B: 240, A: 255}, nil
	case "hotpink":
		return sdl.Color{R: 255, G: 105, B: 180, A: 255}, nil
	case "indianred":
		return sdl.Color{R: 205, G: 92, B: 92, A: 255}, nil
	case "indigo":
		return sdl.Color{R: 75, G: 0, B: 130, A: 255}, nil
	case "ivory":
		return sdl.Color{R: 255, G: 255, B: 240, A: 255}, nil
	case "khaki":
		return sdl.Color{R: 240, G: 230, B: 140, A: 255}, nil
	case "lavender":
		return sdl.Color{R: 230, G: 230, B: 250, A: 255}, nil
	case "lavenderblush":
		return sdl.Color{R: 255, G: 240, B: 245, A: 255}, nil
	case "lawngreen":
		return sdl.Color{R: 124, G: 252, B: 0, A: 255}, nil
	case "lemonchiffon":
		return sdl.Color{R: 255, G: 250, B: 205, A: 255}, nil
	case "lightblue":
		return sdl.Color{R: 173, G: 216, B: 230, A: 255}, nil
	case "lightcoral":
		return sdl.Color{R: 240, G: 128, B: 128, A: 255}, nil
	case "lightcyan":
		return sdl.Color{R: 224, G: 255, B: 255, A: 255}, nil
	case "lightgoldenrodyellow":
		return sdl.Color{R: 250, G: 250, B: 210, A: 255}, nil
	case "lightgray":
		return sdl.Color{R: 211, G: 211, B: 211, A: 255}, nil
	case "lightgreen":
		return sdl.Color{R: 144, G: 238, B: 144, A: 255}, nil
	case "lightgrey":
		return sdl.Color{R: 211, G: 211, B: 211, A: 255}, nil
	case "lightpink":
		return sdl.Color{R: 255, G: 182, B: 193, A: 255}, nil
	case "lightsalmon":
		return sdl.Color{R: 255, G: 160, B: 122, A: 255}, nil
	case "lightseagreen":
		return sdl.Color{R: 32, G: 178, B: 170, A: 255}, nil
	case "lightskyblue":
		return sdl.Color{R: 135, G: 206, B: 250, A: 255}, nil
	case "lightslategray":
		return sdl.Color{R: 119, G: 136, B: 153, A: 255}, nil
	case "lightslategrey":
		return sdl.Color{R: 119, G: 136, B: 153, A: 255}, nil
	case "lightsteelblue":
		return sdl.Color{R: 176, G: 196, B: 222, A: 255}, nil
	case "lightyellow":
		return sdl.Color{R: 255, G: 255, B: 224, A: 255}, nil
	case "lime":
		return sdl.Color{R: 0, G: 255, B: 0, A: 255}, nil
	case "limegreen":
		return sdl.Color{R: 50, G: 205, B: 50, A: 255}, nil
	case "linen":
		return sdl.Color{R: 250, G: 240, B: 230, A: 255}, nil
	case "magenta":
		return sdl.Color{R: 255, G: 0, B: 255, A: 255}, nil
	case "maroon":
		return sdl.Color{R: 128, G: 0, B: 0, A: 255}, nil
	case "mediumaquamarine":
		return sdl.Color{R: 102, G: 205, B: 170, A: 255}, nil
	case "mediumblue":
		return sdl.Color{R: 0, G: 0, B: 205, A: 255}, nil
	case "mediumorchid":
		return sdl.Color{R: 186, G: 85, B: 211, A: 255}, nil
	case "mediumpurple":
		return sdl.Color{R: 147, G: 112, B: 219, A: 255}, nil
	case "mediumseagreen":
		return sdl.Color{R: 60, G: 179, B: 113, A: 255}, nil
	case "mediumslateblue":
		return sdl.Color{R: 123, G: 104, B: 238, A: 255}, nil
	case "mediumspringgreen":
		return sdl.Color{R: 0, G: 250, B: 154, A: 255}, nil
	case "mediumturquoise":
		return sdl.Color{R: 72, G: 209, B: 204, A: 255}, nil
	case "mediumvioletred":
		return sdl.Color{R: 199, G: 21, B: 133, A: 255}, nil
	case "midnightblue":
		return sdl.Color{R: 25, G: 25, B: 112, A: 255}, nil
	case "mintcream":
		return sdl.Color{R: 245, G: 255, B: 250, A: 255}, nil
	case "mistyrose":
		return sdl.Color{R: 255, G: 228, B: 225, A: 255}, nil
	case "moccasin":
		return sdl.Color{R: 255, G: 228, B: 181, A: 255}, nil
	case "navajowhite":
		return sdl.Color{R: 255, G: 222, B: 173, A: 255}, nil
	case "navy":
		return sdl.Color{R: 0, G: 0, B: 128, A: 255}, nil
	case "oldlace":
		return sdl.Color{R: 253, G: 245, B: 230, A: 255}, nil
	case "olive":
		return sdl.Color{R: 128, G: 128, B: 0, A: 255}, nil
	case "olivedrab":
		return sdl.Color{R: 107, G: 142, B: 35, A: 255}, nil
	case "orange":
		return sdl.Color{R: 255, G: 165, B: 0, A: 255}, nil
	case "orangered":
		return sdl.Color{R: 255, G: 69, B: 0, A: 255}, nil
	case "orchid":
		return sdl.Color{R: 218, G: 112, B: 214, A: 255}, nil
	case "palegoldenrod":
		return sdl.Color{R: 238, G: 232, B: 170, A: 255}, nil
	case "palegreen":
		return sdl.Color{R: 152, G: 251, B: 152, A: 255}, nil
	case "paleturquoise":
		return sdl.Color{R: 175, G: 238, B: 238, A: 255}, nil
	case "palevioletred":
		return sdl.Color{R: 219, G: 112, B: 147, A: 255}, nil
	case "papayawhip":
		return sdl.Color{R: 255, G: 239, B: 213, A: 255}, nil
	case "peachpuff":
		return sdl.Color{R: 255, G: 218, B: 185, A: 255}, nil
	case "peru":
		return sdl.Color{R: 205, G: 133, B: 63, A: 255}, nil
	case "pink":
		return sdl.Color{R: 255, G: 192, B: 203, A: 255}, nil
	case "plum":
		return sdl.Color{R: 221, G: 160, B: 221, A: 255}, nil
	case "powderblue":
		return sdl.Color{R: 176, G: 224, B: 230, A: 255}, nil
	case "purple":
		return sdl.Color{R: 128, G: 0, B: 128, A: 255}, nil
	case "rebeccapurple":
		return sdl.Color{R: 102, G: 51, B: 153, A: 255}, nil
	case "red":
		return sdl.Color{R: 255, G: 0, B: 0, A: 255}, nil
	case "rosybrown":
		return sdl.Color{R: 188, G: 143, B: 143, A: 255}, nil
	case "royalblue":
		return sdl.Color{R: 65, G: 105, B: 225, A: 255}, nil
	case "saddlebrown":
		return sdl.Color{R: 139, G: 69, B: 19, A: 255}, nil
	case "salmon":
		return sdl.Color{R: 250, G: 128, B: 114, A: 255}, nil
	case "sandybrown":
		return sdl.Color{R: 244, G: 164, B: 96, A: 255}, nil
	case "seagreen":
		return sdl.Color{R: 46, G: 139, B: 87, A: 255}, nil
	case "seashell":
		return sdl.Color{R: 255, G: 245, B: 238, A: 255}, nil
	case "sienna":
		return sdl.Color{R: 160, G: 82, B: 45, A: 255}, nil
	case "silver":
		return sdl.Color{R: 192, G: 192, B: 192, A: 255}, nil
	case "skyblue":
		return sdl.Color{R: 135, G: 206, B: 235, A: 255}, nil
	case "slateblue":
		return sdl.Color{R: 106, G: 90, B: 205, A: 255}, nil
	case "slategray":
		return sdl.Color{R: 112, G: 128, B: 144, A: 255}, nil
	case "slategrey":
		return sdl.Color{R: 112, G: 128, B: 144, A: 255}, nil
	case "snow":
		return sdl.Color{R: 255, G: 250, B: 250, A: 255}, nil
	case "springgreen":
		return sdl.Color{R: 0, G: 255, B: 127, A: 255}, nil
	case "steelblue":
		return sdl.Color{R: 70, G: 130, B: 180, A: 255}, nil
	case "tan":
		return sdl.Color{R: 210, G: 180, B: 140, A: 255}, nil
	case "teal":
		return sdl.Color{R: 0, G: 128, B: 128, A: 255}, nil
	case "thistle":
		return sdl.Color{R: 216, G: 191, B: 216, A: 255}, nil
	case "tomato":
		return sdl.Color{R: 255, G: 99, B: 71, A: 255}, nil
	case "turquoise":
		return sdl.Color{R: 64, G: 224, B: 208, A: 255}, nil
	case "violet":
		return sdl.Color{R: 238, G: 130, B: 238, A: 255}, nil
	case "wheat":
		return sdl.Color{R: 245, G: 222, B: 179, A: 255}, nil
	case "white":
		return sdl.Color{R: 255, G: 255, B: 255, A: 255}, nil
	case "whitesmoke":
		return sdl.Color{R: 245, G: 245, B: 245, A: 255}, nil
	case "yellow":
		return sdl.Color{R: 255, G: 255, B: 0, A: 255}, nil
	case "yellowgreen":
		return sdl.Color{R: 154, G: 205, B: 50, A: 255}, nil
	default:
		return sdl.Color{}, ErrInvalidCssValue

	}

}

// <hash-token> of  3, 4, 6, or 8 hexadecimal  digits
// https://drafts.csswg.org/css-color/#hex-notation
func parseHexColor(token css_tokenizer.Token) (sdl.Color, error) {
	hash, ok := token.(*css_tokenizer.MultiCharacterToken)
	if !ok {
		return sdl.Color{}, ErrInvalidCssValue
	}

	s := hash.Value
	switch len(s) {
	case 3: // RGB — expand each nibble: 0xN → 0xNN
		v, err := strconv.ParseUint(s, 16, 16)
		if err != nil {
			return sdl.Color{}, ErrInvalidCssValue
		}
		return sdl.Color{
			R: uint8((v>>8)&0xf) * 0x11,
			G: uint8((v>>4)&0xf) * 0x11,
			B: uint8(v&0xf) * 0x11,
			A: 0xff,
		}, nil
	case 4: // RGBA — expand each nibble
		v, err := strconv.ParseUint(s, 16, 16)
		if err != nil {
			return sdl.Color{}, ErrInvalidCssValue
		}
		return sdl.Color{
			R: uint8((v>>12)&0xf) * 0x11,
			G: uint8((v>>8)&0xf) * 0x11,
			B: uint8((v>>4)&0xf) * 0x11,
			A: uint8(v&0xf) * 0x11,
		}, nil
	case 6: // RRGGBB
		v, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			return sdl.Color{}, ErrInvalidCssValue
		}
		return sdl.Color{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}, nil
	case 8: // RRGGBBAA
		v, err := strconv.ParseUint(s, 16, 64)
		if err != nil {
			return sdl.Color{}, ErrInvalidCssValue
		}
		return sdl.Color{R: uint8(v >> 24), G: uint8(v >> 16), B: uint8(v >> 8), A: uint8(v)}, nil
	default:
		return sdl.Color{}, ErrInvalidCssValue
	}
}
