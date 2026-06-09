package layouts_test

import (
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/css/cssom"
	css_parser "github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/layouts"
	"github.com/VisualSource/plex/internal/layouts/styletree"
	"github.com/VisualSource/plex/internal/utils"
	"github.com/kr/pretty"
)

func TestBuildLayoutTree(t *testing.T) {

	styleNode := createStyleTree(t, `
		div { color: red; height: 100px; }
		div > main {
			color: red;
		}
		main {
			color: green !important;
		}
	`, createElement("div",
		createElement("main"),
	))

	box := layouts.NewLayoutTree(styleNode, &layouts.Context{}, 100, 0)

	t.Log(pretty.Sprintf("%# v", box))
}

func createStyleTree(t *testing.T, input string, el dom.ElementNode) *styletree.StyledNode {
	t.Helper()
	p := css_parser.NewCssParser()

	stylesheet, err := p.ParseStylesheet(strings.NewReader(input), utils.None[string]())
	if err != nil {
		panic("Failed to parse stylesheet")
	}

	list := make([]*css_parser.Stylesheet, 0)
	list = append(list, stylesheet)

	css := cssom.NewCssom(list)

	tree, err := styletree.NewStyleTree(el, css, nil)

	if err != nil {
		panic(err)
	}

	return tree
}

func createElement(tag string, children ...dom.ElementNode) dom.ElementNode {
	el := dom.NewElement(nil, tag, utils.None[dom.Namespace](), utils.None[string](), utils.None[string](), false, utils.None[string](), nil)

	for _, child := range children {
		el.AppendChild(child)
	}

	return el
}
