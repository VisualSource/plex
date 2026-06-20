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

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/text"
	"github.com/VisualSource/plex/internal/css/cssom"
	css_parser "github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/css/selector"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/html_parser"
	"github.com/VisualSource/plex/internal/layouts"
	"github.com/VisualSource/plex/internal/layouts/styletree"
	"github.com/VisualSource/plex/internal/layouts/widgets"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type FrameConfig struct{}
type scriptEvent struct {
	Type string
}
type Frame struct {
	width, height int

	logger *slog.Logger

	document *dom.Document
	cssom    *cssom.Cssom

	paint  *styletree.StyledNode
	layout *layouts.Box

	// used by script engine to parse selectors and other stuff
	cssParser *css_parser.CssParser

	rw sync.RWMutex

	ctx context.Context

	shaper *text.Shaper
	state  *widgets.WidgetState

	wasm []api.Module

	scriptChannal chan scriptEvent

	scriptRuntime wazero.Runtime
}

func NewFrame(logger *slog.Logger, ctx context.Context) *Frame {
	shaper := text.NewShaper(text.WithCollection(gofont.Collection()))

	runtime := wazero.NewRuntime(ctx)

	return &Frame{
		scriptChannal: make(chan scriptEvent),
		scriptRuntime: runtime,
		ctx:           ctx,
		logger:        logger,
		cssom:         &cssom.Cssom{},
		shaper:        shaper,
		state:         widgets.NewWidgetState(shaper),
		cssParser:     css_parser.NewCssParser(),
	}
}

func (f *Frame) LoadRemote(url url.URL, allowRelativeFileImport bool) error {
	switch url.Scheme {
	case "file":
		if !filepath.IsAbs(url.Path) {
			return errors.New("Invalid file path")
		}

		file, err := os.OpenFile(url.Path, os.O_RDONLY, 0666)
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
	Source             string
	BaseUrl            *url.URL
	NoScript           bool
	Favicon            string
	RelativeFileImport bool
}

func (m *Metadata) ParsePath(value string) (*url.URL, error) {
	//Need to handle resloving relative paths useing baseURL

	uri, err := url.Parse(value)
	if err != nil {
		return nil, err
	}

	return uri, nil
}

func (f *Frame) Load(stream io.Reader) error {
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

	client := &http.Client{}
	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)

	envBuilder := f.scriptRuntime.NewHostModuleBuilder("env").NewFunctionBuilder()
	printFn := api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
		mem := mod.Memory()
		offset := api.DecodeU32(stack[0])

		strLen, outOfRange := mem.ReadUint32Le(offset)
		if !outOfRange {
			return
		}

		str, outOfRange := mem.Read(offset+4, strLen)
		if !outOfRange {
			return
		}

		f.logger.InfoContext(ctx, string(str))
	})

	_, err = envBuilder.WithGoModuleFunction(printFn, []api.ValueType{api.ValueTypeI32},
		[]api.ValueType{}).Export("print").Instantiate(f.ctx)
	if err != nil {
		f.logger.ErrorContext(f.ctx, err.Error())
	}

	for _, contentItem := range externalContent {
		switch contentItem.Tag() {
		case "script":
			if metadata.NoScript {
				continue
			}
			f.logger.Debug("loading plex script")
			node := contentItem.(dom.ElementNode)

			srcAttribute := node.GetAttribute("src")

			if srcAttribute != nil {
				uri, err := metadata.ParsePath(srcAttribute.Value)
				if err != nil {
					continue
				}

				wg.Go(func() {
					program, err := fetchResource[[]byte](f.ctx, client, uri, parseScript, metadata.RelativeFileImport)
					if err != nil {
						f.logger.Error(err.Error())
						return
					}

					module, err := f.scriptRuntime.Instantiate(f.ctx, program)
					if err != nil {
						f.logger.Error(err.Error())
						return
					}

					mu.Lock()
					f.wasm = append(f.wasm, module)
					mu.Unlock()
				})
			} else {

				scriptText := getTextContent(node)
				if scriptText == "" {
					f.logger.Debug("empty script content")
					continue
				}

				wg.Go(func() {
					program, err := parseScript(strings.NewReader(scriptText))
					if err != nil {
						f.logger.Error("failed to parse script", slog.Any("error", err))
						return
					}

					module, err := f.scriptRuntime.Instantiate(f.ctx, program)
					if err != nil {
						f.logger.Error(err.Error())
						return
					}

					mu.Lock()
					f.wasm = append(f.wasm, module)
					mu.Unlock()
				})
			}
		case "style":
			node := contentItem.(dom.ElementNode)

			styleText := getTextContent(node)
			if styleText == "" {
				continue
			}

			wg.Go(func() {

				sheet, err := parseCss(strings.NewReader(styleText))
				if err != nil {
					f.logger.Error(err.Error())
					return
				}

				mu.Lock()
				f.cssom.AppendStylesheet(sheet)
				mu.Unlock()
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
						result, err := fetchResource(f.ctx, client, uri, parseCss, metadata.RelativeFileImport)
						if err != nil {

							return
						}
						mu.Lock()
						f.cssom.AppendStylesheet(result)
						mu.Unlock()
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
			f.logger.WarnContext(f.ctx, "unhandled tag", slog.String("tag", contentItem.Tag()))
		}
	}

	wg.Wait()

	root, err := selector.QuerySelector(f.document, "html")
	if err != nil {
		return err
	}

	tree, err := styletree.NewStyleTree(root.(dom.ElementNode), f.cssom, nil)
	if err != nil {
		return err
	}

	f.paint = tree
	f.layout = layouts.NewLayoutTree(f.paint, &layouts.Context{Shaper: f.shaper}, float64(f.width), float64(f.height))

	go func() {
		// env state
		f.logger.Debug("Starting script event loop")

		for _, module := range f.wasm {
			exports := module.ExportedFunctionDefinitions()
			if _, ok := exports["main"]; ok {
				if _, err := module.ExportedFunction("main").Call(f.ctx); err != nil {
					f.logger.ErrorContext(f.ctx, err.Error())
				}
			}
		}

		for {
			select {
			case <-f.ctx.Done():
				return
			case message := <-f.scriptChannal:
				switch message.Type {

				}
			}
		}
	}()

	return nil
}

func (f *Frame) Destroy() {
	close(f.scriptChannal)
	f.scriptRuntime.Close(f.ctx)
}

func (f *Frame) Render(gtx layout.Context) {
	if f.layout == nil {
		return
	}

	widgets.RenderTree(gtx, f.layout, f.state)
}
func (f *Frame) Resize(width, height int) error {
	f.width = width
	f.height = height
	//TODO: should not have to though away all of the styletree and layout

	root, err := selector.QuerySelector(f.document, "html")
	if err != nil {
		return err
	}

	tree, err := styletree.NewStyleTree(root.(dom.ElementNode), f.cssom, nil)
	if err != nil {
		return err
	}

	ly := layouts.NewLayoutTree(tree, &layouts.Context{Shaper: f.shaper}, float64(width), float64(height))

	f.paint = tree
	f.layout = ly

	return nil
}
