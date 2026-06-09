package styletree

import (
	"cmp"
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/css/cssom"
	css_parser "github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/css/selector"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/utils"
)

type PropertyMap map[string]*cssom.Declaration

func GetProp[T comparable](propMap PropertyMap, prop string) utils.Option[T] {
	if value, ok := propMap[prop]; ok {
		if decValue, ok := value.Value.(T); ok {
			return utils.Some(decValue)
		}
	}

	return utils.None[T]()
}

func GetPropOrDefault[T any](style *StyledNode, prop string, def T) T {
	if style == nil {
		return def
	}

	if pv, ok := style.SpecifiedValues[prop]; ok {
		if value, ok := pv.Value.(T); ok {
			return value
		}

	}
	return def
}

type StyledNode struct {
	Element         dom.Node
	SpecifiedValues PropertyMap
	Children        []*StyledNode
}

func NewStyleTree(el dom.ElementNode, css *cssom.Cssom, parentPropertyMap PropertyMap) (*StyledNode, error) {

	props := specifiedValues(el, css, parentPropertyMap)

	if v, ok := props["display"].Value.(cssom.Display); ok && v.Outer == "none" {
		return nil, nil
	}

	children := make([]*StyledNode, 0)
	for _, child := range el.Children() {
		if n, ok := child.(dom.ElementNode); ok {
			tree, err := NewStyleTree(n, css, props)

			if err != nil {
				continue
			}

			if tree == nil {
				continue
			}

			children = append(children, tree)
		} else if child, ok := child.(*dom.Text); ok {
			textNode := &StyledNode{
				Element:         child,
				Children:        make([]*StyledNode, 0),
				SpecifiedValues: inheritProperties(make(PropertyMap), props),
			}

			children = append(children, textNode)
		}
	}

	node := &StyledNode{
		Element:         el,
		Children:        children,
		SpecifiedValues: props,
	}

	return node, nil
}

func matchRules(node dom.ElementNode, css *cssom.Cssom) []*cssom.StyleRule {

	matched := make([]*cssom.StyleRule, 0)

	for _, stylesheet := range css.Stylesheets {
		for _, rule := range stylesheet.Rules {
			switch r := rule.(type) {
			case *cssom.StyleRule:
				if r.Selector.Matches(node) {
					matched = append(matched, r)
				}
			}
		}
	}

	return matched
}

func specifiedValues(node dom.ElementNode, css *cssom.Cssom, parentPropertyMap PropertyMap) PropertyMap {
	rules := matchRules(node, css)
	style := node.GetAttribute("style")
	if style != nil {
		p := css_parser.NewCssParser()
		values, err := p.ParseBlocksContents(strings.NewReader(style.Value))
		if err == nil && len(values) > 0 {
			// should only be a single declaration list and no rules
			// as it should only be parsing something like style="color: green; background-color:gray;"

			list, ok := values[0].(*css_parser.DeclarationList)
			if ok {
				block, err := cssom.ParseDeclarationList(list)
				if err == nil {
					rules = append(rules, &cssom.StyleRule{
						Selector: &selector.Selector{
							Specificity: 1000,
						},
						Declarations: block,
						Stylesheet: &cssom.Stylesheet{
							Origin: 2, //TODO: use const defined value
						},
					})
				}
			}
		}
	}

	// need to deal with declarations Important values
	slices.SortStableFunc(rules, func(a, b *cssom.StyleRule) int {
		return cmp.Or(
			cmp.Compare(a.Stylesheet.Origin, b.Stylesheet.Origin),
			cmp.Compare(a.Selector.Specificity, b.Selector.Specificity),
		)
	})

	property := make(PropertyMap)
	for _, rule := range rules {
		for key, value := range rule.Declarations {
			property[key] = value
		}
	}

	property = inheritProperties(property, parentPropertyMap)

	return property
}
