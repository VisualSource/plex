package cssom

import (
	"errors"

	css_parser "github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/css/selector"
	"github.com/VisualSource/plex/internal/utils"
)

type Cssom struct {
	Rules []Rule
}

func NewCssom(stylesheets []*css_parser.Stylesheet) *Cssom {

	rules := make([]Rule, 0)
	for _, stylesheet := range stylesheets {
		for _, rule := range stylesheet.Rules {
			rule, err := parseRule(rule)
			if err == nil {
				rules = append(rules, rule)
			}
		}
	}

	return &Cssom{
		Rules: rules,
	}
}

type Rule interface{}

type StyleRule struct {
	Origin       int
	Selector     selector.Selector
	Declarations DeclarationBlock
}

func parseRule(rule *css_parser.Rule) (Rule, error) {

	_, err := parseDeclarationList(rule.Declarations)
	if err != nil {
		return nil, err
	}

	return &StyleRule{}, nil
}

type DeclarationBlock map[string]*Declaration

func parseDeclarationList(list *css_parser.DeclarationList) (DeclarationBlock, error) {
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

var ErrUnsupportedProperty = errors.New("unsupported property")

func parseDeclaration(properties DeclarationBlock, declaration *css_parser.Declaration) DeclarationBlock {
	name := css_parser.GetTokenValueAsString(declaration.Name)

	switch name {
	case "background":
		// inherits

	case "border":
		// inherits

	case "padding":
		// inherits

	case "margin":
		//inherits

	case "margin-inline", "margin-block":

	case "margin-left", "margin-right", "margin-top", "margin-bottom", "margin-block-end", "margin-block-start", "margin-inline-end", "margin-inline-start":
		value, err := parseMargin(declaration.Value)
		if err != nil {
			break
		}

		properties[name] = &Declaration{
			Important: declaration.Important,
			CssText:   declaration.OriginalText,
			Value:     value,
		}
	case "position":

	case "height", "width":
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
