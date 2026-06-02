package core

import (
	"strings"

	"github.com/VisualSource/plex/internal/css/cssom"
	css_parser "github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/html_parser"
	"github.com/VisualSource/plex/internal/layouts"
	"github.com/VisualSource/plex/internal/layouts/styletree"
	"github.com/VisualSource/plex/internal/utils"
)

func renderHtml(html string, style string, width, height float32) *layouts.Box {

	hp := html_parser.NewHtmlParser(strings.NewReader(html))
	doc, err := hp.Parse()
	if err != nil {
		panic(err)
	}

	css := css_parser.NewCssParser()

	sheet, err := css.ParseStylesheet(strings.NewReader(style), utils.None[string]())

	if err != nil {
		panic(err)
	}

	cssom := cssom.NewCssom([]*css_parser.Stylesheet{sheet})

	root := doc.Children()[1].(dom.ElementNode)

	stylet, err := styletree.NewStyleTree(root, cssom, nil)
	if err != nil {
		panic(err)
	}

	return layouts.NewLayoutTree(stylet, width, height)
}
