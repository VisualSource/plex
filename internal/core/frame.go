package core

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gioui.org/layout"
	"github.com/VisualSource/plex/internal/css/cssom"
	css_parser "github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/css/selector"
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

func (f *Frame) LoadRemote(url url.URL) error {

	var wg sync.WaitGroup
	switch url.Scheme {
	case "file":
		if !filepath.IsAbs(url.Path) {
			return errors.New("Invalid file path")
		}

		file, err := os.OpenFile(url.Path, os.O_RDONLY, os.ModeDevice)
		if err != nil {
			return err
		}
		defer file.Close()

		hp := html_parser.NewHtmlParser(file)

		doc, err := hp.Parse()
		if err != nil {
			return err
		}
		f.document = doc
		// pull out script,css and other metadata

		content, err := selector.QuerySelectorAll(doc, "script,style,link,meta,title")
		if err != nil {
			return err
		}

		seenTitle := false
		pageTitle := url.Path

		list := make([]*css_parser.Stylesheet, 0)

		client := &http.Client{}

		for _, item := range content {
			switch item.Tag() {
			case "script":
				wg.Go(func() {
					//TODO: parse, and compile
				})
			case "style":
				wg.Go(func() {
					style := item.Children()[0].(*dom.Text)
					sheet, err := f.cssParser.ParseStylesheet(strings.NewReader(style.Data), utils.None[string]())
					if err != nil {
						f.logger.ErrorContext(f.ctx, "failed to parse stylesheet", slog.String("error", err.Error()))
						return
					}
					list = append(list, sheet)
				})
			case "link":
				node := item.(dom.ElementNode)

				target := node.GetAttribute("rel")

				if target != nil {
					switch target.Value {
					case "stylesheet":
						href := node.GetAttribute("href")
						if href == nil {
							continue
						}
						path, err := url.Parse(href.Value)
						if err != nil {
							f.logger.ErrorContext(f.ctx, "failed to parse url", slog.String("error", err.Error()))
							continue
						}

						wg.Go(func() {
							resp, err := client.Get(path.RequestURI())
							if err != nil {
								f.logger.ErrorContext(f.ctx, "failed to make http request", slog.String("error", err.Error()))
								return
							}
							defer resp.Body.Close()

							cssp := css_parser.NewCssParser()

							st, err := cssp.ParseStylesheet(resp.Body, utils.Some(path.RequestURI()))
							if err != nil {
								f.logger.ErrorContext(f.ctx, "failed to make http request", slog.String("error", err.Error()))
								return
							}

							list = append(list, st)
						})
					}
				}
			case "meta":
			case "title":
				if seenTitle {
					continue
				}
				seenTitle = true
				title := item.Children()[0].(*dom.Text)
				pageTitle = title.Data
			}
		}

		f.cssom = cssom.NewCssom(list)
	case "https", "http":
		return errors.ErrUnsupported
	default:
		return errors.New("unable to load file!")
	}

	wg.Wait()

	return nil
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
