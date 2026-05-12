package html_parser

import "github.com/VisualSource/plex/internal/dom"

type activeFormattingItem struct {
	IsMarker bool
	Element  dom.Node
}

func newActiveFormattingMarker() activeFormattingItem {
	return activeFormattingItem{IsMarker: true}
}
