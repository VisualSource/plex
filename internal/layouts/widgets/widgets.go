package widgets

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/VisualSource/plex/internal/css/cssom"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/layouts"
	"github.com/VisualSource/plex/internal/layouts/styletree"
)

type WidgetState struct {
	widgets map[dom.Node]LayoutWidget
	lt      *text.Shaper
}

// NewWidgetState constructs a WidgetState. The shaper must be the same instance
// used by the layout pass so measurement and rendering agree on glyph metrics.
func NewWidgetState(shaper *text.Shaper) *WidgetState {
	return &WidgetState{
		widgets: make(map[dom.Node]LayoutWidget, 0),
		lt:      shaper,
	}
}

type LayoutWidget interface {
	Layout(gtx layout.Context, box *layouts.Box) layout.Dimensions
}

var defaultFontSizeProp = cssom.Size{
	Kind: cssom.SizeKindLP,
	LP:   &cssom.LengthPercentage{Value: layouts.DefaultFontSize, Unit: "px"},
}

func RenderTree(gtx layout.Context, box *layouts.Box, state *WidgetState) {
	bb := box.Dimensions.BorderBox()

	boxOffset := op.Offset(image.Pt(int(bb.X), int(bb.Y))).Push(gtx.Ops)

	if !box.Anonymous {
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
		case "#text":
			renderText(gtx, box, state)
		default:
			RenderBox(gtx, box)
		}
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

func renderText(gtx layout.Context, box *layouts.Box, state *WidgetState) {
	fontSize := layouts.DefaultFontSize
	if fs := styletree.GetPropOrDefault(box.Style, "font-size", defaultFontSizeProp); fs.Kind == cssom.SizeKindLP && fs.LP.Value > 0 {
		fontSize = fs.LP.Value
	}

	// Leave gtx.Constraints alone: widget.Label uses Constraints.Max.X as the
	// shaper's MaxWidth and will wrap if that's narrower than the natural text
	// width. Truncating Content.W to int can put it just below the value Label
	// computes internally, which would wrap every glyph to its own line.
	// Positioning of the glyphs comes from the op.Offset pushes already on the
	// stack, so we don't need to constrain here.
	gtx.Constraints.Min = image.Point{}

	textColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: color.NRGBA{A: 0xFF}}.Add(gtx.Ops)
	textColor := textColorMacro.Stop()

	// Layout shaped text in CSS pixels. widget.Label internally converts
	// `size` via gtx.Sp(...) before shaping, so we feed it the Sp value that
	// reverses that conversion — keeping the shaped size identical to what
	// measureText used (fixed.I(int(fontSize)) pixels-per-em).
	spSize := unit.Sp(float32(fontSize))
	if gtx.Metric.PxPerSp > 0 {
		spSize = unit.Sp(float32(fontSize) / gtx.Metric.PxPerSp)
	}

	widget.Label{}.Layout(
		gtx,
		state.lt,
		layouts.DefaultTextFont,
		spSize,
		box.TextContent,
		textColor,
	)
}
