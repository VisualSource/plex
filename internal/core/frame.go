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

type FrameConfig struct{}

type Frame struct {
	width, height int

	logger *slog.Logger

	document *dom.Document
	cssom    *cssom.Cssom

	paint  *styletree.StyledNode
	layout *layouts.Box

	// used by script engine to parse selectors and other stuff
	cssParser *css_parser.CssParser
}

func NewFrame(logger *slog.Logger) *Frame {
	return &Frame{
		logger:    logger,
		cssom:     &cssom.Cssom{},
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

func (f *Frame) LoadRemote(url url.URL, allowRelativeFileImport bool) error {
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
	Title              string
	BaseUrl            *url.URL
	NoScript           bool
	Favicon            string
	RelativeFileImport bool
}

func (m *Metadata) ParsePath(value string) (*url.URL, error) {
	if m.BaseUrl != nil && m.BaseUrl.Scheme == "file" {

	}

	uri, err := url.Parse(value)
	if err != nil {
		return nil, err
	}

	if !uri.IsAbs() {
		// TODO resolve against baseurl
	}

	return uri, nil
}

func fetchResource[T any](ctx context.Context, client *http.Client, url *url.URL, handler func(io.Reader) (T, error), allowRelativeFileImport bool) (T, error) {
	var zero T

	switch url.Scheme {
	case "file":
		if !allowRelativeFileImport {
			return zero, errors.New("not allowed to load local resource")
		}

		file, err := os.OpenFile(url.Path, os.O_RDONLY, 0666 /* read/write access for everyone */)
		if err != nil {
			return zero, err
		}
		defer file.Close()

		result, err := handler(file)
		if err != nil {
			return zero, err
		}

		return result, nil
	case "http", "https":
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
		if err != nil {
			return zero, nil
		}

		resp, err := client.Do(req)
		if err != nil {
			return zero, err
		}
		defer resp.Body.Close()

		result, err := handler(resp.Body)
		if err != nil {
			return zero, err
		}

		return result, nil
	default:
		return zero, errors.ErrUnsupported
	}
}

func getTextContent(node dom.ElementNode) string {
	children := node.Children()
	if len(children) == 0 {
		return ""
	}

	if text, ok := children[0].(*dom.Text); ok {
		return text.Data
	}

	return ""
}

func (f *Frame) Load(stream io.Reader) error {

	ctx, _ := context.WithCancel(context.Background())
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
				uri, err := metadata.ParsePath(srcAttribute.Value)
				if err != nil {
					continue
				}

				wg.Go(func() {
					_, err := fetchResource(ctx, client, uri, func(r io.Reader) (any, error) {
						return nil, nil
					}, metadata.RelativeFileImport)
					if err != nil {
						return
					}

					//TODO: parse and compile script

				})
			} else {
				scriptText := getTextContent(node)
				if scriptText == "" {
					continue
				}

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

			styleText := getTextContent(node)
			if styleText == "" {
				continue
			}

			wg.Go(func() {
				select {
				case <-ctx.Done():
				default:
					sheet, err := parseCss(strings.NewReader(styleText))
					if err != nil {
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
					uri, err := metadata.ParsePath(href.Value)
					if err != nil {
						continue
					}

					wg.Go(func() {
						result, err := fetchResource(ctx, client, uri, parseCss, metadata.RelativeFileImport)
						if err != nil {

							return
						}

						stylesheetsChan <- result

					})
				case "icon":

				}
			}
		case "meta":
		case "title":
			if seenTitle {
				continue
			}
			seenTitle = true
			metadata.Title = getTextContent(contentItem.(dom.ElementNode))

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
