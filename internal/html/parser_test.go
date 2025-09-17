package html_test

import (
	"strings"
	"testing"

	html "github.com/VisualSource/plex/internal/html"
)

func TestHtmlParser(t *testing.T) {
	stream := strings.NewReader("<html></html>")

	parser := html.NewHTMLParser(stream)

	err := parser.Parse()

	if err != nil {
		t.Error("Failed to parse input")
	}
}
