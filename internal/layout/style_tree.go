package layout

import (
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/dom"
)

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

func NewStyleTree(el dom.ElementNode, stylesheets []*parser.Stylesheet, selectorCache map[*parser.Rule]*Selector) (*StyledNode, error) {
	if selectorCache == nil {
		selectorCache = make(map[*parser.Rule]*Selector)
	}

	children := make([]*StyledNode, 0)
	for _, child := range el.Children() {
		if n, ok := child.(dom.ElementNode); ok {
			tree, err := NewStyleTree(n, stylesheets, selectorCache)

			if err != nil {
				continue
			}

			children = append(children, tree)

		}
	}

	props := specifiedValues(el, stylesheets, selectorCache)
	node := &StyledNode{
		node:            el,
		children:        children,
		specifiedValues: props,
	}

	return node, nil
}

func matchRule(node dom.ElementNode, stylesheetOrigin int, rule *parser.Rule, selectorCache map[*parser.Rule]*Selector) *MatchedRule {
	selector, ok := selectorCache[rule]
	if !ok {
		sel, err := NewSelectorFromTokens(rule.Prelude)
		if err != nil {
			return nil
		}

		selectorCache[rule] = sel
	}

	if selector.Matches(node) {
		return &MatchedRule{
			Specificity: selector.Specificity,
			Rule:        rule,
			Origin:      stylesheetOrigin,
		}
	}

	return nil
}

func matchRules(node dom.ElementNode, stylesheets []*parser.Stylesheet, selectorCache map[*parser.Rule]*Selector) []*MatchedRule {

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

func specifiedValues(node dom.ElementNode, stylesheets []*parser.Stylesheet, selectorCache map[*parser.Rule]*Selector) map[string]any {
	property := make(map[string]any)

	rules := matchRules(node, stylesheets, selectorCache)

	slices.SortFunc(rules, func(a, b *MatchedRule) int {
		if a.Origin < b.Origin {
			return -1
		}

		return a.Specificity - b.Specificity
	})

	for _, rule := range rules {
		for _, dec := range rule.Rule.Declarations.Value {
			// apply props
		}
	}

	style := node.GetAttribute("style")
	if style != nil {
		p := parser.NewCssParser()
		_, err := p.ParseBlocksContents(strings.NewReader(style.Value))
		if err == nil {
			// should only get declaration lists
		}
	}

	return property
}
