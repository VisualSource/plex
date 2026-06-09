package layouts

import (
	"github.com/VisualSource/plex/internal/css/cssom"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/layouts/styletree"
)

type Edge struct {
	Top, Left, Right, Bottom float64
}

type Rect struct {
	H, W, X, Y float64
}

type Dimensions struct {
	Content                 Rect
	Padding, Border, Margin Edge
}

func (d *Dimensions) PaddingBox() Rect {
	return Rect{
		X: d.Content.X - d.Padding.Left,
		Y: d.Content.Y - d.Padding.Top,
		W: d.Content.W + d.Padding.Left + d.Padding.Right,
		H: d.Content.H + d.Padding.Top + d.Padding.Bottom,
	}
}
func (d *Dimensions) BorderBox() Rect {
	pb := d.PaddingBox()

	pb.X -= d.Border.Left
	pb.Y -= d.Border.Top

	pb.W += d.Border.Left + d.Border.Right
	pb.H += d.Border.Top + d.Border.Bottom

	return pb
}
func (d *Dimensions) MarginBox() Rect {
	bb := d.BorderBox()

	bb.X -= d.Margin.Left
	bb.Y -= d.Margin.Top

	bb.W += d.Margin.Left + d.Margin.Right
	bb.H += d.Margin.Top + d.Margin.Bottom

	return bb
}

type OuterBoxType uint
type InnerBoxType uint

const (
	OuterBoxType_Block OuterBoxType = iota
	OuterBoxType_Inline
	OuterBoxType_RunIn
)

const (
	InnerBoxType_Flow InnerBoxType = iota
	InnerBoxType_FlowRoot
)

type Box struct {
	Anonymous  bool
	Dimensions Dimensions
	OuterType  OuterBoxType
	InnerType  InnerBoxType
	Style      *styletree.StyledNode
	Children   []*Box
}

func (b *Box) getInlineContainer() *Box {
	switch b.OuterType {
	case OuterBoxType_Block:
		if len(b.Children) >= 1 && b.Children[len(b.Children)-1].Anonymous {
			return b.Children[len(b.Children)-1]
		} else {
			box := NewAnonymousBox()

			b.Children = append(b.Children, box)

			return box
		}
	default:
		return b
	}
}

func (b *Box) calculateDimensions(dimensions *Dimensions) {
	b.calculateWidth(dimensions)
	b.calculatePosition(dimensions)

	if b.isInlineContext() {
		b.calculateInlineChildDimensions()
	} else {
		b.calculateChildDimensions()
	}

	b.calculateHeight(dimensions)
}

const (
	field_MarginLeft int = iota
	field_MarginRight
	field_BorderLeft
	field_BorderRight
	field_PaddingLeft
	field_PaddingRight
	field_Width
)

func resolveSize(size *cssom.Size, parentInlineSize float64) (float64, bool) {
	switch size.Kind {
	case cssom.SizeKindKeyword:
		if size.Keyword == "auto" {
			return 0, true
		}
	case cssom.SizeKindLP:
		switch size.LP.Unit {
		case "px":
			return size.LP.Value, false
		case "%":
			return parentInlineSize * (size.LP.Value / 100), false
		}
	}

	return 0, false
}

var defaultZero = cssom.Size{
	Kind: cssom.SizeKindLP,
	LP: &cssom.LengthPercentage{
		Value: 0,
		Unit:  "px",
	},
}
var defaultAuto = cssom.Size{
	Kind:    cssom.SizeKindKeyword,
	Keyword: "auto",
}

func resolveOrZero(prop string, node *styletree.StyledNode, parentSize float64) float64 {
	data := styletree.GetPropOrDefault(node.SpecifiedValues, prop, defaultZero)

	value, _ := resolveSize(&data, parentSize)

	return value
}

func (b *Box) calculateWidth(parent *Dimensions) {
	mb := parent.MarginBox()

	switch b.OuterType {
	case OuterBoxType_Inline:

		// inline boxes don't stretch — margins, borders, padding are all resolved,
		// but width comes from content (set by measureInlineSize, called by parent)
		b.Dimensions.Padding.Left = resolveOrZero("padding-left", b.Style, mb.W)
		b.Dimensions.Padding.Right = resolveOrZero("padding-right", b.Style, mb.W)
		b.Dimensions.Border.Left = resolveOrZero("border-left-width", b.Style, mb.W)
		b.Dimensions.Border.Right = resolveOrZero("border-right-width", b.Style, mb.W)
		b.Dimensions.Margin.Left = resolveOrZero("margin-left", b.Style, mb.W)

	case OuterBoxType_Block:
		sizes := [7]cssom.Size{
			styletree.GetPropOrDefault(b.Style.SpecifiedValues, "margin-left", defaultZero),  // initial 0
			styletree.GetPropOrDefault(b.Style.SpecifiedValues, "margin-right", defaultZero), // initial0

			styletree.GetPropOrDefault(b.Style.SpecifiedValues, "border-left-width", defaultZero),  // resolve border-left-style on none|hidden compute value as zero else use value. initial value is medium which is 3px
			styletree.GetPropOrDefault(b.Style.SpecifiedValues, "border-right-width", defaultZero), // resolve border-right-style on none|hidden compute value as zero else use value. initial value is medium which is 3px

			styletree.GetPropOrDefault(b.Style.SpecifiedValues, "padding-left", defaultZero),  // initial 0
			styletree.GetPropOrDefault(b.Style.SpecifiedValues, "padding-right", defaultZero), // initial 0

			styletree.GetPropOrDefault(b.Style.SpecifiedValues, "width", defaultAuto), // initial  auto, should set before hand in styletree building?
		}

		var total float64
		values := make([]float64, 7)
		autoFields := make([]bool, 7)
		for i, item := range sizes {
			value, isAuto := resolveSize(&item, mb.W)

			if isAuto {
				autoFields[i] = true
				continue
			}

			values[i] = value
			total += value
		}

		//#region resolve auto
		underflow := parent.Content.W - total

		switch {
		case !autoFields[field_Width] && !autoFields[field_MarginLeft] && !autoFields[field_MarginRight]:
			values[field_MarginRight] += underflow

		case !autoFields[field_Width] && !autoFields[field_MarginLeft] && autoFields[field_MarginRight]:
			values[field_MarginRight] = underflow
		case !autoFields[field_Width] && autoFields[field_MarginLeft] && !autoFields[field_MarginRight]:
			values[field_MarginLeft] = underflow
		case autoFields[field_Width]:
			if underflow >= 0.0 {
				values[field_Width] = underflow
			} else {
				values[field_Width] = 0
				values[field_MarginRight] += underflow
			}
		case !autoFields[field_Width] && autoFields[field_MarginLeft] && autoFields[field_MarginRight]:
			values[field_MarginLeft] = underflow / 2.0
			values[field_MarginRight] = underflow / 2.0
		}

		//#endregion

		// TODO: apply min/max width clamping
		//TODO: apply intrinsic keywords max-content/min-content
		//TODO: apply aspect ratio

		b.Dimensions.Content.W = values[field_Width]
		b.Dimensions.Padding.Left = values[field_PaddingLeft]
		b.Dimensions.Padding.Right = values[field_PaddingRight]
		b.Dimensions.Border.Left = values[field_BorderLeft]
		b.Dimensions.Border.Right = values[field_BorderRight]
		b.Dimensions.Margin.Left = values[field_MarginLeft]
		b.Dimensions.Margin.Right = values[field_MarginRight]
	}
}

const (
	field_MarginTop int = iota
	field_MarginBottom
	field_BorderTop
	field_BorderBottom
	field_PaddingTop
	field_PaddingBottom
)

func (b *Box) calculatePosition(parent *Dimensions) {
	mb := parent.MarginBox()
	switch b.OuterType {
	case OuterBoxType_Inline:
		// margin-top/bottom have no effect on inline elements (CSS spec)

		// Position is set by the parent's line box algorithm, not here.
		// Only resolve vertical spacing.
		b.Dimensions.Border.Top = resolveOrZero("border-top-width", b.Style, mb.H)
		b.Dimensions.Border.Bottom = resolveOrZero("border-bottom-width", b.Style, mb.H)
		b.Dimensions.Padding.Top = resolveOrZero("padding-top", b.Style, mb.H)
		b.Dimensions.Padding.Bottom = resolveOrZero("padding-bottom", b.Style, mb.H)
	//@see  http://www.w3.org/TR/CSS2/visudet.html#normal-block
	case OuterBoxType_Block:
		sizes := [6]cssom.Size{
			styletree.GetPropOrDefault(b.Style.SpecifiedValues, "margin-top", defaultZero),
			styletree.GetPropOrDefault(b.Style.SpecifiedValues, "margin-bottom", defaultZero),

			styletree.GetPropOrDefault(b.Style.SpecifiedValues, "border-top-width", defaultZero),    // resolve border-top-style on none|hidden compute value as zero else use value. initial value is medium which is 3px
			styletree.GetPropOrDefault(b.Style.SpecifiedValues, "border-bottom-width", defaultZero), // resolve border-bottom-style on none|hidden compute value as zero else use value. initial value is medium which is 3px

			styletree.GetPropOrDefault(b.Style.SpecifiedValues, "padding-top", defaultZero),
			styletree.GetPropOrDefault(b.Style.SpecifiedValues, "padding-bottom", defaultZero),
		}

		values := make([]float64, 6)
		autoFields := make([]bool, 6)
		for i, item := range sizes {
			value, isAuto := resolveSize(&item, mb.H)

			if isAuto {
				autoFields[i] = true
				continue
			}

			values[i] = value
		}

		b.Dimensions.Margin.Top = values[field_MarginTop]
		b.Dimensions.Margin.Bottom = values[field_MarginBottom]

		b.Dimensions.Border.Top = values[field_BorderTop]
		b.Dimensions.Border.Bottom = values[field_BorderBottom]

		b.Dimensions.Padding.Top = values[field_PaddingTop]
		b.Dimensions.Padding.Bottom = values[field_PaddingBottom]

		b.Dimensions.Content.X = b.Dimensions.Margin.Left + b.Dimensions.Border.Left + b.Dimensions.Padding.Left
		b.Dimensions.Content.Y = parent.Content.H + b.Dimensions.Margin.Top + b.Dimensions.Border.Top + b.Dimensions.Padding.Top

	}
}

func (b *Box) calculateHeight(parent *Dimensions) {
	switch b.OuterType {
	case OuterBoxType_Block:
		heightSize := styletree.GetProp[cssom.Size](b.Style.SpecifiedValues, "height")
		if heightSize.IsNone() {
			break
		}

		mb := parent.MarginBox()
		height, isAuto := resolveSize(heightSize.Value, mb.H)
		if isAuto {
			break
		}

		b.Dimensions.Content.H = height
	}
}

func (b *Box) calculateInlineChildDimensions() {
	var lineX, lineY, lineHeight float64

	for _, child := range b.Children {
		child.measureInlineSize()

		marginW := child.Dimensions.Margin.Left + child.Dimensions.Margin.Right +
			child.Dimensions.Border.Left + child.Dimensions.Border.Right +
			child.Dimensions.Padding.Left + child.Dimensions.Padding.Right

		marginH := child.Dimensions.Margin.Top + child.Dimensions.Margin.Bottom +
			child.Dimensions.Border.Top + child.Dimensions.Border.Bottom +
			child.Dimensions.Padding.Top + child.Dimensions.Padding.Bottom

		childW := child.Dimensions.Content.W + marginW
		childH := child.Dimensions.Content.H + marginH

		if lineX+childW > b.Dimensions.Content.W && lineX > 0 {
			lineY += lineHeight
			lineX = 0
			lineHeight = 0
		}

		child.Dimensions.Content.X = lineX + child.Dimensions.Margin.Left +
			child.Dimensions.Border.Left + child.Dimensions.Padding.Left
		child.Dimensions.Content.Y = lineY + child.Dimensions.Margin.Top +
			child.Dimensions.Border.Top + child.Dimensions.Padding.Top

		lineX += childW
		if childH > lineHeight {
			lineHeight = childH
		}
	}

	b.Dimensions.Content.H += lineY + lineHeight

}

func (b *Box) measureInlineSize() {
	fontSize := 16.0
	if fs := styletree.GetPropOrDefault(b.Style.SpecifiedValues, "font-size", defaultZero); fs.Kind == cssom.SizeKindLP {
		fontSize = fs.LP.Value
	}

	if textNode, ok := b.Style.Element.(*dom.Text); ok {
		// Stub: estimate width as characters × 60% of font size
		b.Dimensions.Content.W = float64(len([]rune(textNode.Data))) * fontSize * 0.6
		b.Dimensions.Content.H = fontSize
		return
	}

	// Inline element with children: sum child widths, max child heights
	var totalW, maxH float64
	for _, child := range b.Children {
		child.measureInlineSize()
		totalW += child.Dimensions.Content.W
		if child.Dimensions.Content.H > maxH {
			maxH = child.Dimensions.Content.H
		}
	}
	b.Dimensions.Content.W = totalW
	b.Dimensions.Content.H = maxH
}

func (b *Box) calculateChildDimensions() {
	for _, child := range b.Children {

		child.calculateDimensions(&b.Dimensions)

		mb := child.Dimensions.MarginBox()
		b.Dimensions.Content.H += mb.H
	}
}

func (b *Box) isInlineContext() bool {
	for _, child := range b.Children {
		if child.OuterType == OuterBoxType_Inline {
			return true
		}
	}
	return false
}

func NewAnonymousBox() *Box {
	return &Box{
		Anonymous: true,
		OuterType: OuterBoxType_Block,
		InnerType: InnerBoxType_Flow,
		Children:  make([]*Box, 0),
	}
}
