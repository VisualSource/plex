package widgets

import (
	"gioui.org/layout"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/layouts"
)

type WidgetState struct {
	button *Button
	input  *TextInput
}

func RenderTree(gtx layout.Context, box *layouts.Box, state *WidgetState) {

	switch box.Style.Element.Tag() {
	case "button":
		if state.button == nil {
			state.button = &Button{}
		}

		state.button.Layout(gtx, box)
	case "input":
		typeAttr := box.Style.Element.(dom.ElementNode).GetAttribute("type")
		if typeAttr != nil {
			switch typeAttr.Value {
			case "text":
				if state.input == nil {
					state.input = NewTextInput()
				}
				state.input.Layout(gtx, box)
			}
		} else {
			if state.input == nil {
				state.input = NewTextInput()
			}
			state.input.Layout(gtx, box)
		}
	default:
		RenderBox(gtx, box)
	}

	for _, child := range box.Children {
		RenderTree(gtx, child, state)
	}
}
