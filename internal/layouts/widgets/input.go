package widgets

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/VisualSource/plex/internal/layouts"
	"github.com/VisualSource/plex/internal/layouts/styletree"
)

type TextInput struct {
	editor widget.Editor
	shaper *text.Shaper
	font   font.Font
}

func NewTextInput() *TextInput {
	return &TextInput{
		editor: widget.Editor{
			SingleLine: true,
		},
		shaper: text.NewShaper(text.WithCollection(gofont.Collection())),
		font: font.Font{
			Typeface: "Times New Roman, Georgia, serif",
			Style:    font.Regular,
			Weight:   0,
		},
	}
}

func (i *TextInput) Layout(gtx layout.Context, box *layouts.Box) layout.Dimensions {
	bb := box.Dimensions.BorderBox()

	size := image.Rect(int(bb.X), int(bb.Y), int(bb.X+bb.W), int(bb.Y+bb.H))

	for {
		_, ok := i.editor.Update(gtx)
		if !ok {
			break
		}
	}

	bgC := color.NRGBA{}
	if bgColor := styletree.GetProp[color.NRGBA](box.Style.SpecifiedValues, "background-color"); bgColor.IsSome() {
		bgC = *bgColor.Value
	}

	fgC := color.NRGBA{A: 0xFF}
	if bgColor := styletree.GetProp[color.NRGBA](box.Style.SpecifiedValues, "color"); bgColor.IsSome() {
		fgC = *bgColor.Value
	}

	defer clip.Rect{Max: size.Max, Min: size.Min}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: bgC}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	textColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: fgC}.Add(gtx.Ops)
	textColor := textColorMacro.Stop()

	selectionColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: color.NRGBA{B: 0x80, A: 0xAA}}.Add(gtx.Ops)
	selectionColor := selectionColorMacro.Stop()

	i.editor.Layout(gtx, i.shaper, i.font, unit.Sp(16), textColor, selectionColor)

	return layout.Dimensions{Size: size.Size()}
}
