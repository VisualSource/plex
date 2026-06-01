package styletree_test

import (
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/css/cssom"
	css_parser "github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/layouts/styletree"
	"github.com/VisualSource/plex/internal/utils"
	"github.com/kr/pretty"
)

func TestNewStyleTree(t *testing.T) {

	stylesheets := createStylesheet(t, `
		@media screen and (width >= 900px) {
			main {
				width: 100%;
			}
		}
		* { display: block; }
		div { color: red; }
		div > main {
			color: red;
		}
		main {
			color: green !important;
		}
	`)
	el := createElement("div",
		createElement("main"),
	)
	styleTree, err := styletree.NewStyleTree(el, stylesheets, nil)
	if err != nil {
		t.Fatalf("failed to parse styleTree %s", err)
	}

	t.Log(pretty.Sprintf("%# v", styleTree))
}

func createStylesheet(t *testing.T, input string) *cssom.Cssom {
	t.Helper()
	p := css_parser.NewCssParser()

	stylesheet, err := p.ParseStylesheet(strings.NewReader(input), utils.None[string]())
	if err != nil {
		panic("Failed to parse stylesheet")
	}

	list := make([]*css_parser.Stylesheet, 0)
	list = append(list, stylesheet)

	return cssom.NewCssom(list)
}

func createElement(tag string, children ...dom.ElementNode) dom.ElementNode {
	el := dom.NewElement(nil, tag, utils.None[dom.Namespace](), utils.None[string](), utils.None[string](), false, utils.None[string](), nil)

	for _, child := range children {
		el.AppendChild(child)
	}

	return el
}
