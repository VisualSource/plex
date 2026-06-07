package widgets

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/VisualSource/plex/internal/layouts"
	"github.com/VisualSource/plex/internal/layouts/styletree"
)

func RenderBox(gtx layout.Context, box *layouts.Box) layout.Dimensions {
	bb := box.Dimensions.BorderBox()

	area := clip.Rect{
		Min: image.Pt(int(bb.X), int(bb.Y)),
		Max: image.Pt(int(bb.X+bb.W), int(bb.Y+bb.H)),
	}

	defer area.Push(gtx.Ops).Pop()

	if bgColor := styletree.GetProp[color.NRGBA](box.Style.SpecifiedValues, "background-color"); bgColor.IsSome() {
		paint.ColorOp{Color: *bgColor.Value}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}

	return layout.Dimensions{
		Size: image.Point{
			area.Max.X - area.Min.X,
			area.Max.Y - area.Min.Y,
		}}
}
