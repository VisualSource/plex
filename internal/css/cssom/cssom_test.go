package cssom_test

import (
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/css/cssom"
	css_parser "github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/utils"
	"github.com/kr/pretty"
)

func TestParseColor(t *testing.T) {
	sheets := createStylesheet(t, `
		* { color: red; }
	`)

	css := cssom.NewCssom(sheets)

	t.Log(pretty.Sprintf("%# v", css))
}

func TestParseMedia(t *testing.T) {
	sheets := createStylesheet(t, `
		@media screen and (width >= 900px) {
			article {
				padding: 1rem 3rem;
			}
		}
	`)

	css := cssom.NewCssom(sheets)

	t.Log(pretty.Sprintf("%# v", css))
}

func createStylesheet(t *testing.T, input string) []*css_parser.Stylesheet {
	t.Helper()
	p := css_parser.NewCssParser()

	stylesheet, err := p.ParseStylesheet(strings.NewReader(input), utils.None[string]())
	if err != nil {
		panic("Failed to parse stylesheet")
	}

	list := make([]*css_parser.Stylesheet, 0)
	list = append(list, stylesheet)

	return list
}
