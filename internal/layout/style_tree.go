package layout

import (
	"github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/dom"
)

type StyledNode struct {
	node            dom.Node
	specifiedValues map[string]any
	children        []*StyledNode
}

func NewStyleTree(dom *dom.Document, stylesheets *parser.Stylesheet) (*StyledNode, error) {

	return nil, nil
}

func matchRule(node dom.Node, rule *parser.Rule) any {
	return nil
}

func matchRules(node dom.Node, stylesheet *parser.Stylesheet) []any {
	return nil
}

func specifiedValues(node dom.Node, stylesheet *parser.Stylesheet) {}
