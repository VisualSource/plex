package widgets

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/layouts"
)

type WidgetState struct {
	widgets map[dom.Node]LayoutWidget
}

type LayoutWidget interface {
	Layout(gtx layout.Context, box *layouts.Box) layout.Dimensions
}

func RenderTree(gtx layout.Context, box *layouts.Box, state *WidgetState) {
	bb := box.Dimensions.BorderBox()

	boxOffset := op.Offset(image.Pt(int(bb.X), int(bb.Y))).Push(gtx.Ops)

	switch box.Style.Element.Tag() {
	case "button":
		if btn, ok := state.widgets[box.Style.Element]; ok {
			btn.Layout(gtx, box)
		} else {
			btn := &Button{}
			state.widgets[box.Style.Element] = btn
			btn.Layout(gtx, box)
		}

	case "input":
		if input, ok := state.widgets[box.Style.Element]; ok {
			input.Layout(gtx, box)
		} else if elNode, ok := box.Style.Element.(dom.ElementNode); ok {
			typeAttr := elNode.GetAttribute("type")

			switch {
			case typeAttr != nil && typeAttr.Value == "hidden":
			case typeAttr != nil && typeAttr.Value == "password":
				input := NewTextInput()
				input.editor.Mask = '*'

				state.widgets[elNode] = input
				input.Layout(gtx, box)
			default:
				input := NewTextInput()
				state.widgets[elNode] = input
				input.Layout(gtx, box)
			}
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
