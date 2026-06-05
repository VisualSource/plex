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

	// used by script engine to parse selectors and other stuff
	cssParser *css_parser.CssParser
}

func NewFrame(logger *slog.Logger, ctx context.Context) *Frame {
	return &Frame{
		ctx:       ctx,
		logger:    logger,
		cssParser: css_parser.NewCssParser(),
	}
}

func parseCss(reader io.Reader) (*cssom.Stylesheet, error) {
	p := css_parser.NewCssParser()
	sheet, err := p.ParseStylesheet(reader, utils.None[string]())
	if err != nil {

		return nil, err
	}

	cssomSheet := cssom.ParseStylesheet(sheet)

	//TODO: load linked stylesheet via @import

	return cssomSheet, nil
}

func (f *Frame) LoadRemote(url url.URL, allowResolveResourceLocal bool) error {
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

		return f.Load(file)

	case "https", "http":
		resp, err := http.Get(url.String())
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return f.Load(resp.Body)
	default:
		return errors.ErrUnsupported
	}

}

type Metadata struct {
	Title    string
	BaseUrl  string
	NoScript bool
}

func (m *Metadata) ParseURL(value string) (*url.URL, error) {
	uri, err := url.Parse(value)
	if err != nil {
		return nil, err
	}

	if !uri.IsAbs() {
		// TODO resolve against baseurl
	}

	return uri, nil
}

func (f *Frame) Load(stream io.Reader) error {

	ctx, _ := context.WithCancel(f.ctx)
	metadata := &Metadata{}

	hp := html_parser.NewHtmlParser(stream)

	doc, err := hp.Parse()
	if err != nil {
		return err
	}
	f.document = doc

	externalContent, err := selector.QuerySelectorAll(doc, "script,style,link,meta,title,base")
	if err != nil {
		return err
	}

	seenTitle := false

	stylesheetsChan := make(chan *cssom.Stylesheet, 0)
	client := &http.Client{}
	var wg sync.WaitGroup

	for _, contentItem := range externalContent {
		switch contentItem.Tag() {
		case "script":
			if !metadata.NoScript {
				continue
			}

			node := contentItem.(dom.ElementNode)

			srcAttribute := node.GetAttribute("src")

			if srcAttribute != nil {
				uri, err := metadata.ParseURL(srcAttribute.Value)
				if err != nil {
					continue
				}

				wg.Go(func() {
					req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), nil)
					if err != nil {
						return
					}

					resp, err := client.Do(req)
					if err != nil {
						return
					}
					defer resp.Body.Close()

					//TODO: parse and compile script

				})
			} else {
				scriptText := node.Children()[0].(*dom.Text)

				wg.Go(func() {
					select {
					case <-ctx.Done():

					default:
						//TODO: parse and compile script

					}
				})
			}
		case "style":
			node := contentItem.(dom.ElementNode)

			styleText := node.Children()[0].(*dom.Text).Data

			wg.Go(func() {
				select {
				case <-ctx.Done():
				default:
					sheet, err := parseCss(strings.NewReader(styleText))
					if err != nil {

						f.logger.ErrorContext(f.ctx, "failed to parse stylesheet", slog.String("error", err.Error()))
						return
					}

					stylesheetsChan <- sheet
				}
			})
		case "link":
			node := contentItem.(dom.ElementNode)

			if rel := node.GetAttribute("rel"); rel != nil {
				switch rel.Value {
				case "stylesheet":
					href := node.GetAttribute("href")
					if href == nil {
						continue
					}
					uri, err := metadata.ParseURL(href.Value)
					if err != nil {
						continue
					}

					wg.Go(func() {
						req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), nil)
						if err != nil {
							return
						}

						resp, err := client.Do(req)
						if err != nil {
							return
						}
						defer resp.Body.Close()

						sheet, err := parseCss(resp.Body)
						if err != nil {

							f.logger.ErrorContext(f.ctx, "failed to make http request", slog.String("error", err.Error()))
							return
						}

						stylesheetsChan <- sheet

					})
				}
			}
		case "meta":
		case "title":
			if seenTitle {
				continue
			}
			seenTitle = true
			title := contentItem.Children()[0].(*dom.Text)
			metadata.Title = title.Data
		default:
			f.logger.WarnContext(ctx, "unhandled tag", slog.String("tag", contentItem.Tag()))
		}
	}

	wg.Wait()

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

/*

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
	`

*/
