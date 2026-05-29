package styletree

import (
	"cmp"
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/css/tokenizer"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/layout/selector"
	"github.com/Zyko0/go-sdl3/sdl"
)

type PropertyMap map[string]any

func (p PropertyMap) MustGetColor(key string) sdl.FColor {
	color, ok := p[key].(sdl.FColor)
	if !ok {
		panic("was expecting a color value")
	}

	return color
}

func (p PropertyMap) MustGetString(key string) string {
	color, ok := p[key].(string)
	if !ok {
		panic("was expecting a string value")
	}

	return color
}

type SelectorCache map[*parser.Rule]*selector.Selector

type MatchedRule struct {
	Specificity int
	Rule        *parser.Rule
	Origin      int
}
type StyledNode struct {
	node            dom.Node
	specifiedValues map[string]any
	children        []*StyledNode
}

func NewStyleTree(el dom.ElementNode, stylesheets []*parser.Stylesheet, selectorCache SelectorCache, parentPropertyMap PropertyMap) (*StyledNode, error) {
	if selectorCache == nil {
		selectorCache = make(SelectorCache)
	}

	props := specifiedValues(el, stylesheets, selectorCache, parentPropertyMap)

	display, hasDisplay := props["display"]

	if hasDisplay && display.(string) != "none" {
		return nil, nil
	}

	children := make([]*StyledNode, 0)
	for _, child := range el.Children() {
		if n, ok := child.(dom.ElementNode); ok {
			tree, err := NewStyleTree(n, stylesheets, selectorCache, props)

			if err != nil {
				continue
			}

			if tree == nil {
				continue
			}

			children = append(children, tree)
		} else if _, ok := child.(*dom.Text); ok {

			//TODO: handle passing text nodes styles

		}
	}

	node := &StyledNode{
		node:            el,
		children:        children,
		specifiedValues: props,
	}

	return node, nil
}

func matchRule(node dom.ElementNode, stylesheetOrigin int, rule *parser.Rule, selectorCache SelectorCache) *MatchedRule {
	sele, ok := selectorCache[rule]
	if !ok {
		sel, err := selector.NewSelectorFromTokens(rule.Prelude)
		if err != nil {
			return nil
		}

		selectorCache[rule] = sel
		sele = sel
	}

	if sele.Matches(node) {
		return &MatchedRule{
			Specificity: sele.Specificity,
			Rule:        rule,
			Origin:      stylesheetOrigin,
		}
	}

	return nil
}

func matchRules(node dom.ElementNode, stylesheets []*parser.Stylesheet, selectorCache SelectorCache) []*MatchedRule {

	matched := make([]*MatchedRule, 0)

	for _, stylesheet := range stylesheets {
		for _, rule := range stylesheet.Rules {
			m := matchRule(node, stylesheet.Origin, rule, selectorCache)
			if m != nil {
				matched = append(matched, m)
			}
		}
	}

	return matched
}

func parseDeclaration(propertyMap PropertyMap, parentPropertyMap PropertyMap, decl *parser.Declaration) map[string]any {

	//background, border, margin, padding, width, height, display, and position

	switch decl.Name.IsToken() {
	case tokenizer.TokenId_Ident:
		key := parser.GetTokenValueAsString(decl.Name)
		switch key {
		//#region shorthands
		case "background": // explist inherit
		case "border": // explist inherit
		case "padding": // explist inherit
		case "margin": //explist inherit

		//#endregion
		case "width":
		case "hight":
		case "display":
		case "position":

		case "color":
			// inherit,initial,revert,unset,currentColor
			// fn,hex,named,

			propertyMap[key] = parseColor(decl.Value)

		case "background-color":

		default:
			propertyMap[key] = ""
		}
	}

	return propertyMap
}

func specifiedValues(node dom.ElementNode, stylesheets []*parser.Stylesheet, selectorCache SelectorCache, parentPropertyMap PropertyMap) PropertyMap {
	property := make(PropertyMap)

	property = inheritProperties(property, parentPropertyMap)

	rules := matchRules(node, stylesheets, selectorCache)

	slices.SortStableFunc(rules, func(a, b *MatchedRule) int {
		return cmp.Or(
			cmp.Compare(a.Origin, b.Origin),
			cmp.Compare(a.Specificity, b.Specificity),
		)
	})

	for _, rule := range rules {
		for _, dec := range rule.Rule.Declarations.Value {
			property = parseDeclaration(property, parentPropertyMap, dec)
		}
	}

	style := node.GetAttribute("style")
	if style != nil {
		p := parser.NewCssParser()
		values, err := p.ParseBlocksContents(strings.NewReader(style.Value))
		if err == nil && len(values) > 0 {
			// should only be a single declaration list and no rules
			// as it should only be parsing something like style="color: green; background-color:gray;"

			list, ok := values[0].(*parser.DeclarationList)
			if ok {
				for _, dec := range list.Value {
					property = parseDeclaration(property, parentPropertyMap, dec)
				}
			}
		}
	}

	return property
}
