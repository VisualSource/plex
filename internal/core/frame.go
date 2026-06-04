package core

import (
	"context"
	"io"
	"log/slog"
	"strings"

	"gioui.org/layout"
	"github.com/VisualSource/plex/internal/css/cssom"
	css_parser "github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/html_parser"
	"github.com/VisualSource/plex/internal/layouts"
	"github.com/VisualSource/plex/internal/layouts/styletree"
	"github.com/VisualSource/plex/internal/layouts/widgets"
	"github.com/VisualSource/plex/internal/utils"
)

type Frame struct {
	width, height int

	ctx context.Context

	logger *slog.Logger

	document *dom.Document
	cssom    *cssom.Cssom

	paint  *styletree.StyledNode
	layout *layouts.Box

	// used by script engine to parse stuff
	cssParser *css_parser.CssParser
}

func NewFrame(logger *slog.Logger, ctx context.Context) *Frame {
	return &Frame{
		ctx:       ctx,
		logger:    logger,
		cssParser: css_parser.NewCssParser(),
	}
}

func (f *Frame) Load(stream io.Reader) error {
	htmlParser := html_parser.NewHtmlParser(stream)
	doc, err := htmlParser.Parse()
	if err != nil {
		return err
	}

	f.document = doc

	// TODO:
	// scripts and css should be load via speclive html parser
	// just parse for now
	userAgentStylesheet, err := f.cssParser.ParseStylesheet(strings.NewReader( // TODO: move this some where else
		`div { display: block; padding-left: 12px; padding-right: 12px; padding-top: 12px; padding-bottom: 12px; }
	head { display: none; background-color: gray; }
	html { display: block; background-color: maroon; }
	body { display: block; background-color: coral; }
	.a { background-color: #ff0000; }
	.b { background-color: #ffa500; }
	.c { background-color: #ffff00; }
	.d { background-color: #008000; }
	.e { background-color: #0000ff; }
	.f { background-color: #4b0082; }
	.g { background-color: #800080; }
	`), utils.Some("userAgent"))
	if err != nil {
		return err
	}

	f.cssom = cssom.NewCssom([]*css_parser.Stylesheet{userAgentStylesheet})

	var root dom.ElementNode
	for _, child := range doc.Children() {
		if el, ok := child.(dom.ElementNode); ok {
			root = el
			break
		}
	}

	if root != nil {
		t, err := styletree.NewStyleTree(root, f.cssom, nil)
		if err != nil {
			return err
		}

		f.paint = t

		f.layout = layouts.NewLayoutTree(f.paint, float64(f.width), float64(f.height))
	}

	return nil
}

func (f *Frame) Render(gtx layout.Context) {
	if f.layout == nil {
		return
	}

	widgets.RenderTree(gtx, f.layout)
}
func (f *Frame) Resize(width, height int) {
	f.width = width
	f.height = height
	//TODO: should not have to though away all of the styletree and layout

	//TODO update styletree and layout
}
