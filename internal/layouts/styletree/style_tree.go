package styletree

import (
	"cmp"
	"image/color"
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/css/cssom"
	css_parser "github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/css/selector"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/utils"
)

type PropertyMap map[string]*cssom.Declaration

func (t PropertyMap) GetPropColor(prop string) utils.Option[color.NRGBA] {
	if value, ok := t[prop]; ok {
		if decValue, ok := value.Value.(color.NRGBA); ok {
			return utils.Some(decValue)
		}
	}

	return utils.None[color.NRGBA]()
}

func (t PropertyMap) GetPropAsString(prop string) utils.Option[string] {
	if value, ok := t[prop]; ok {
		if decValue, ok := value.Value.(string); ok {
			return utils.Some(decValue)
		}
	}
	return utils.None[string]()
}

func (t PropertyMap) GetPropAsDisplay(prop string) utils.Option[cssom.Display] {
	if value, ok := t[prop]; ok {
		if decValue, ok := value.Value.(cssom.Display); ok {
			return utils.Some(decValue)
		}
	}

	return utils.None[cssom.Display]()
}

func (t PropertyMap) GetPropAsSize(prop string) utils.Option[cssom.Size] {
	if value, ok := t[prop]; ok {
		if decValue, ok := value.Value.(cssom.Size); ok {
			return utils.Some(decValue)
		}
	}

	return utils.None[cssom.Size]()
}

type StyledNode struct {
	Element         dom.Node
	SpecifiedValues PropertyMap
	Children        []*StyledNode
}

func NewStyleTree(el dom.ElementNode, css *cssom.Cssom, parentPropertyMap PropertyMap) (*StyledNode, error) {

	props := specifiedValues(el, css, parentPropertyMap)

	if v, ok := props["display"]; ok && v.IsStringValue("none") {
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
		} else if _, ok := child.(*dom.Text); ok {

			//TODO: handle passing text nodes styles

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
