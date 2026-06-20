package core

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/VisualSource/plex/internal/css/cssom"
	css_parser "github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/script"
	"github.com/VisualSource/plex/internal/script/compiler"
	"github.com/VisualSource/plex/internal/utils"
)

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

func parseScript(reader io.Reader) ([]byte, error) {
	program, err := script.Parse(reader)
	if err != nil {
		return nil, err
	}

	wat, err := compiler.CompileProgram(program)
	if err != nil {
		return nil, err
	}

	tempDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tempDir, "script.*.wat")
	if err != nil {
		return nil, err
	}
	if _, err := tmpFile.WriteString(wat); err != nil {
		return nil, err
	}

	inputFile := tmpFile.Name()
	outputFile := strings.Replace(inputFile, "wat", "wasm", 1)

	cmd := exec.Command("wat2wasm", tmpFile.Name(), "-o", outputFile) // when able replace this with binrary wasm so we do have to pass it to wat2wasm
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Join(err, errors.New(string(cmdOutput)))
	}
	wasm, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, err
	}

	return wasm, nil
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
