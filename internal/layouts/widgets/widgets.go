package widgets

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/VisualSource/plex/internal/layouts"
)

func ColorBox(gtx layout.Context, size image.Point, color color.NRGBA) layout.Dimensions {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: color}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

func RenderTree(gtx layout.Context, box *layouts.Box) {
	met := unit.Metric{}

	bb := box.Dimensions.BorderBox()

	dpX := met.PxToDp(int(bb.X))
	dpY := met.PxToDp(int(bb.Y))
	dpH := met.PxToDp(int(bb.H))
	dpW := met.PxToDp(int(bb.W))

	rect := clip.Rect{
		Min: image.Pt(int(dpW), int(dpX)),
		Max: image.Pt(int(dpY), int(dpH)),
	}.Push(gtx.Ops)

	if bgColor := box.Style.SpecifiedValues.GetPropColor("background-color"); bgColor.IsSome() {
		paint.ColorOp{Color: *bgColor.Value}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}

	rect.Pop()

	for _, child := range box.Children {
		RenderTree(gtx, child)
	}
}
