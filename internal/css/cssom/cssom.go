package cssom

import (
	"image/color"

	css_parser "github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/css/selector"
	css_tokenizer "github.com/VisualSource/plex/internal/css/tokenizer"
	"github.com/VisualSource/plex/internal/utils"
)

type Cssom struct {
	Stylesheets []*Stylesheet
}

func NewCssom(stylesheets []*css_parser.Stylesheet) *Cssom {

	styles := make([]*Stylesheet, 0)
	for _, stylesheet := range stylesheets {
		styles = append(styles, ParseStylesheet(stylesheet))
	}

	return &Cssom{
		Stylesheets: styles,
	}
}

func (c *Cssom) AppendStylesheet(stylesheet *Stylesheet) {
	c.Stylesheets = append(c.Stylesheets, stylesheet)
}

type Stylesheet struct {
	Rules    []Rule
	Location utils.StringOption
	Origin   int
}

func ParseStylesheet(stylesheet *css_parser.Stylesheet) *Stylesheet {
	sheet := &Stylesheet{
		Location: stylesheet.Location,
		Origin:   stylesheet.Origin,
		Rules:    make([]Rule, 0),
	}

	for _, rule := range stylesheet.Rules {
		rule, err := parseRule(rule, sheet)
		if err == nil {
			sheet.Rules = append(sheet.Rules, rule)
		}
	}

	return sheet
}

type Rule interface{}

type StyleRule struct {
	Selector     *selector.Selector
	Declarations DeclarationBlock
	Stylesheet   *Stylesheet
}

func parseRule(rule *css_parser.Rule, stylesheet *Stylesheet) (Rule, error) {
	if rule.Name != "" {
		//TODO: handle @ rules
		return nil, ErrUnsupportedProperty
	}

	sel, err := selector.NewSelectorFromTokens(rule.Prelude)
	if err != nil {
		return nil, err
	}

	value, err := ParseDeclarationList(rule.Declarations)
	if err != nil {
		return nil, err
	}

	return &StyleRule{
		Selector:     sel,
		Declarations: value,
		Stylesheet:   stylesheet,
	}, nil
}

type DeclarationBlock map[string]*Declaration

func ParseDeclarationList(list *css_parser.DeclarationList) (DeclarationBlock, error) {
	properties := make(DeclarationBlock)

	for _, declaration := range list.Value {
		properties = parseDeclaration(properties, declaration)
	}

	return properties, nil
}

type Declaration struct {
	CssText   utils.StringOption
	Important bool
	Value     any
}

func (d *Declaration) IsStringValue(value string) bool {
	if v, ok := d.Value.(string); ok && v == value {
		return true
	}

	return false
}

func (d *Declaration) MustColor() color.NRGBA {
	v, ok := d.Value.(color.NRGBA)
	if !ok {
		panic("unable to get value as color")
	}

	return v
}

func (d *Declaration) MustString() string {
	v, ok := d.Value.(string)
	if !ok {
		panic("unable to get value as string")
	}
	return v
}

func (d *Declaration) MustNumber() *css_tokenizer.NumericToken {
	v, ok := d.Value.(*css_tokenizer.NumericToken)
	if !ok {
		panic("unable to get value as a number")
	}
	return v
}

func parseDeclaration(properties DeclarationBlock, declaration *css_parser.Declaration) DeclarationBlock {
	name := css_parser.GetTokenValueAsString(declaration.Name)

	switch name {

	//#region inherit
	case "font": // shorthand
	case "font-family":
	case "font-size":
		size, err := parseSize(declaration.Value)
		if err != nil {
			break
		}

		properties[name] = &Declaration{
			Important: declaration.Important,
			CssText:   declaration.OriginalText,
			Value:     size,
		}

	case "font-style":
	case "font-variant":
	case "font-weight":
		items := nonWS(declaration.Value)
		if len(items) == 1 {
			value, ok := items[0].(*css_tokenizer.NumericToken) // weights 100-900
			if ok && value.Flag == "integer" && value.Value >= 1 && value.Value < 1000 {
				properties[name] = &Declaration{
					Important: declaration.Important,
					CssText:   declaration.OriginalText,
					Value:     value,
				}
			}
		}
	case "font-size-adjust":
	case "font-stretch":

	case "letter-spacing":
	case "line-height":
		items := nonWS(declaration.Value)
		if len(items) == 1 {
			value, ok := items[0].(*css_tokenizer.NumericToken) // weights 100-900
			if ok {
				properties[name] = &Declaration{
					Important: declaration.Important,
					CssText:   declaration.OriginalText,
					Value:     value,
				}
			}
		}
	case "text-align":
	case "text-indent":
	case "text-shadow":
	case "text-transform":
	case "white-space":
	case "word-break":
	case "word-spacing":
	case "overflow-wrap":
	case "word-wrap":
	case "direction":
	case "unicode-bidi":

	case "list-style": // shothand
	case "list-style-image":
	case "list-style-position":
	case "list-style-type":

	case "border-collapse":
	case "border-spacing":
	case "caption-side":
	case "empty-cells":
	case "cursor":
	case "visibility":
	case "quotes":
	case "orphans":
	case "widows":
	case "page-break-inside":

	//#endregion

	case "background":
	case "border":
	case "margin":
	case "padding":

	case "border-left-width", "border-right-width":
		size, err := parseSize(declaration.Value)
		if err != nil {
			break
		}

		properties[name] = &Declaration{
			Important: declaration.Important,
			CssText:   declaration.OriginalText,
			Value:     size,
		}
	case "margin-top", "margin-right", "margin-bottom", "margin-left":
		size, err := parseSize(declaration.Value)
		if err != nil {
			break
		}

		properties[name] = &Declaration{
			Important: declaration.Important,
			CssText:   declaration.OriginalText,
			Value:     size,
		}
	case "padding-top", "padding-right", "padding-bottom", "padding-left":
		size, err := parseSize(declaration.Value)
		if err != nil {
			break
		}

		properties[name] = &Declaration{
			Important: declaration.Important,
			CssText:   declaration.OriginalText,
			Value:     size,
		}
	case "position":

	case "height", "width", "max-width", "max-height", "min-height", "min-width":
		size, err := parseSize(declaration.Value)
		if err != nil {
			break
		}

		properties[name] = &Declaration{
			Important: declaration.Important,
			CssText:   declaration.OriginalText,
			Value:     size,
		}

	case "background-color", "color":
		color, err := parseColor(declaration.Value)
		if err != nil {
			break
		}

		properties[name] = &Declaration{
			Important: declaration.Important,
			CssText:   declaration.OriginalText,
			Value:     color,
		}
	case "display":
		display, err := parseDisplay(declaration.Value)
		if err != nil {
			break
		}

		properties[name] = &Declaration{
			Important: declaration.Important,
			CssText:   declaration.OriginalText,
			Value:     display,
		}
	default:
		//TODO: css variable check
	}

	return properties
}
