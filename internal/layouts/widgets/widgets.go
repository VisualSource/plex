package widgets

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/VisualSource/plex/internal/layouts"
)

func RenderTree(gtx layout.Context, box *layouts.Box) {

	bb := box.Dimensions.BorderBox()

	x0 := bb.X
	y0 := bb.Y

	x1 := bb.X + bb.W
	y1 := bb.Y + bb.H

	min := image.Pt(int(x0), int(y0))
	max := image.Pt(int(x1), int(y1))

	clip.Rect{
		Min: min,
		Max: max,
	}.Push(gtx.Ops)

	if bgColor := box.Style.SpecifiedValues.GetPropColor("background-color"); bgColor.IsSome() {
		paint.ColorOp{Color: color.NRGBA{R: 0xff, A: 0xFF}}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}

	/*for _, child := range box.Children {
		RenderTree(gtx, child)
	}*/
}
