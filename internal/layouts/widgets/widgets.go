package widgets

import (
	"gioui.org/layout"
	"github.com/VisualSource/plex/internal/layouts"
)

type WidgetState struct {
	button *Button
}

func RenderTree(gtx layout.Context, box *layouts.Box, state *WidgetState) {

	switch box.Style.Element.Tag() {
	case "button":
		if state.button == nil {
			state.button = &Button{}
		}

		state.button.Layout(gtx, box)
	default:
		RenderBox(gtx, box)
	}

	/*bb := box.Dimensions.BorderBox()

	rect := clip.Rect{
		Min: image.Pt(int(bb.X), int(bb.Y)),
		Max: image.Pt(int(bb.X+bb.W), int(bb.Y+bb.H)),
	}.Push(gtx.Ops)

	if bgColor := styletree.GetProp[color.NRGBA](box.Style.SpecifiedValues, "background-color"); bgColor.IsSome() {
		paint.ColorOp{Color: *bgColor.Value}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}

	rect.Pop()*/

	for _, child := range box.Children {
		RenderTree(gtx, child, state)
	}
}
