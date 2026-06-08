package widgets

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/layouts"
)

type WidgetState struct {
	button *Button
	input  *TextInput
}

func RenderTree(gtx layout.Context, box *layouts.Box, state *WidgetState) {
	bb := box.Dimensions.BorderBox()

	boxOffset := op.Offset(image.Pt(int(bb.X), int(bb.Y))).Push(gtx.Ops)

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

	contentOffset := op.Offset(image.Pt(
		int(box.Dimensions.Border.Left+box.Dimensions.Padding.Left),
		int(box.Dimensions.Border.Top+box.Dimensions.Padding.Top),
	)).Push(gtx.Ops)

	for _, child := range box.Children {
		RenderTree(gtx, child, state)
	}

	contentOffset.Pop()
	boxOffset.Pop()
}
