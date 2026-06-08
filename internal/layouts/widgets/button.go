package widgets

import (
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/VisualSource/plex/internal/layouts"
)

type Button struct {
	Pressed bool
	Hover   bool
}

func drawSquare(ops *op.Ops, size image.Rectangle, color color.NRGBA) layout.Dimensions {
	defer clip.Rect{Max: size.Max, Min: size.Min}.Push(ops).Pop()
	paint.ColorOp{Color: color}.Add(ops)
	paint.PaintOp{}.Add(ops)
	return layout.Dimensions{Size: size.Size()}
}
func (b *Button) Layout(gtx layout.Context, box *layouts.Box) layout.Dimensions {
	bb := box.Dimensions.BorderBox()

	size := image.Rect(0, 0, int(bb.W), int(bb.H))

	// Confine the area for pointer events.
	area := clip.Rect(size).Push(gtx.Ops)

	event.Op(gtx.Ops, b)

	// here we loop through all the events associated with this button.
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: b,
			Kinds:  pointer.Press | pointer.Release,
		})
		if !ok {
			break
		}

		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		switch e.Kind {
		case pointer.Press:
			b.Pressed = true
		case pointer.Release:
			b.Pressed = false
		case pointer.Enter:
			b.Hover = true
		case pointer.Leave:
			b.Hover = false
		}
	}

	area.Pop()

	// Draw the button.
	col := color.NRGBA{R: 0x80, A: 0xFF}
	if b.Pressed {
		col = color.NRGBA{G: 0x80, A: 0xFF}
	} else if b.Hover {

		col = color.NRGBA{B: 0x80, A: 0xFF}
	}
	return drawSquare(gtx.Ops, size, col)
	//bb := box.Dimensions.BorderBox()
	/*area := clip.Rect{
		Min: image.Pt(int(bb.X), int(bb.Y)),
		Max: image.Pt(int(bb.X+bb.W), int(bb.Y+bb.H)),
	}

	stack := area.Push(gtx.Ops)
	event.Op(gtx.Ops, b)

	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: b,
		})
		if !ok {
			break
		}

		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		switch e.Kind {
		case pointer.Press:
			b.Pressed = true
		case pointer.Release:
			b.Pressed = false
			//case pointer.Enter:
			//	b.Hover = true
			//case pointer.Leave:
			//	b.Hover = false
		}
	}

	stack.Pop()

	defer clip.Rect{
		Min: image.Pt(0, 0 /*int(bb.X), int(bb.Y)),
		Max: image.Pt(0+100, 0+100 /*int(bb.X+bb.W), int(bb.Y+bb.H)),
	}.Push(gtx.Ops).Pop()

	switch {
	case b.Hover:
		/*if bgColor := styletree.GetProp[color.NRGBA](box.Style.SpecifiedValues, "background-color"); bgColor.IsSome() {
			paint.ColorOp{Color: *bgColor.Value}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
		}
	case b.Pressed:

		paint.ColorOp{Color: color.NRGBA{B: 0x80, A: 0xFF}}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)

	default:
		paint.ColorOp{Color: color.NRGBA{G: 0x80, A: 0xFF}}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		/*if bgColor := styletree.GetProp[color.NRGBA](box.Style.SpecifiedValues, "background-color"); bgColor.IsSome() {
			paint.ColorOp{Color: *bgColor.Value}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
		}

	}

	return layout.Dimensions{Size: image.Point{
		area.Max.X - area.Min.X,
		area.Max.Y - area.Min.Y,
	}}*/
}
