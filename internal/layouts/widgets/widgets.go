package widgets

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/VisualSource/plex/internal/layouts"
)

func RenderTree(gtx layout.Context, box layouts.Box) {
	rect := clip.Rect{
		Min: image.Pt(int(box.Dimensions.Content.X), int(box.Dimensions.Content.Y)),
		Max: image.Pt(int(box.Dimensions.Content.W), int(box.Dimensions.Content.H)),
	}.Push(gtx.Ops)

	bgColor := box.Style.SpecifiedValues.GetPropColor("background-color")
	if bgColor.IsSome() {
		paint.ColorOp{Color: color.NRGBA{R: 0xff, A: 0xFF}}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}

	rect.Pop()

	for _, child := range box.Children {
		RenderTree(gtx, child)
	}
}
