package layouts

import (
	"github.com/VisualSource/plex/internal/css/cssom"
	"github.com/VisualSource/plex/internal/layouts/styletree"
	"github.com/VisualSource/plex/internal/utils"
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
	//b.calculatePosition(dimensions)
	//b.calculateChildDimensions()

	//b.calculateHeight(dimensions)
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

func resloveSize(item *utils.Option[cssom.Size], parent *Dimensions) (bool, float64, bool) {
	if item.IsNone() {
		return true, 0, false
	}
	size := item.Value
	switch size.Kind {
	case cssom.SizeKindKeyword:
		if size.Keyword == "auto" {
			return false, 0, true
		}

	case cssom.SizeKindLP:
		switch size.LP.Unit {
		case "px":
			return false, size.LP.Value, false
		case "%":
			return false, parent.Content.W * (size.LP.Value / 100), false

		}
	}

	return true, 0, false
}

func (b *Box) calculateWidth(parent *Dimensions) {
	switch b.OuterType {
	case OuterBoxType_Block:
		sizes := []utils.Option[cssom.Size]{
			b.Style.SpecifiedValues.GetPropAsSize("margin-left"),
			b.Style.SpecifiedValues.GetPropAsSize("margin-right"),

			b.Style.SpecifiedValues.GetPropAsSize("border-left-width"),
			b.Style.SpecifiedValues.GetPropAsSize("border-right-width"),

			b.Style.SpecifiedValues.GetPropAsSize("padding-left"),
			b.Style.SpecifiedValues.GetPropAsSize("padding-right"),

			b.Style.SpecifiedValues.GetPropAsSize("width"),
		}

		var total float64

		values := make([]float64, 7)
		autodFields := make([]bool, 7)
		for i, item := range sizes {
			isEmpty, value, isAuto := resloveSize(&item, parent)
			if isEmpty {
				continue
			}

			if isAuto {
				autodFields[i] = true
				continue
			}

			values[i] = value
			total += value
		}

		underflow := parent.Content.W - total

		switch {
		case !autodFields[field_Width] && !autodFields[field_MarginLeft] && !autodFields[field_MarginRight]:
			values[field_MarginRight] += underflow
		case !autodFields[field_Width] && !autodFields[field_MarginLeft] && autodFields[field_MarginRight]:
			values[field_MarginRight] = underflow
		case !autodFields[field_Width] && autodFields[field_MarginLeft] && !autodFields[field_MarginRight]:
			values[field_MarginLeft] = underflow
		case autodFields[field_Width]:
			if underflow >= 0.0 {
				values[field_Width] = underflow
			} else {
				values[field_Width] = 0
				values[field_MarginRight] += underflow
			}

		case !autodFields[field_Width] && autodFields[field_MarginLeft] && autodFields[field_MarginRight]:
			values[field_MarginLeft] = underflow / 2.0
			values[field_MarginRight] = underflow / 2.0
		}

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

	switch b.OuterType {
	case OuterBoxType_Block:
		sizes := []utils.Option[cssom.Size]{
			b.Style.SpecifiedValues.GetPropAsSize("margin-top"),
			b.Style.SpecifiedValues.GetPropAsSize("margin-bottom"),

			b.Style.SpecifiedValues.GetPropAsSize("border-top-width"),
			b.Style.SpecifiedValues.GetPropAsSize("border-bottom-width"),

			b.Style.SpecifiedValues.GetPropAsSize("padding-top"),
			b.Style.SpecifiedValues.GetPropAsSize("padding-bottom"),
		}

		values := make([]float64, 6)
		autodFields := make([]bool, 6)
		for i, item := range sizes {
			if item.IsNone() {
				continue
			}
			size := item.Value
			switch size.Kind {
			case cssom.SizeKindKeyword:
				if size.Keyword == "auto" {
					autodFields[i] = true
				}

			case cssom.SizeKindLP:
				switch size.LP.Unit {
				case "px":
					//total += float32(size.LP.Value)

					values[i] = size.LP.Value
				case "%":
					v := parent.Content.H * size.LP.Value / 100
					//total += v
					values[i] = v
				}
			}
		}

		b.Dimensions.Margin.Top = values[field_MarginTop]
		b.Dimensions.Margin.Bottom = values[field_MarginBottom]

		b.Dimensions.Border.Top = values[field_BorderTop]
		b.Dimensions.Border.Bottom = values[field_BorderBottom]

		b.Dimensions.Padding.Top = values[field_PaddingTop]
		b.Dimensions.Padding.Bottom = values[field_PaddingBottom]

		b.Dimensions.Content.X = parent.Content.X + b.Dimensions.Margin.Left + b.Dimensions.Border.Left + b.Dimensions.Padding.Left
		b.Dimensions.Content.Y = parent.Content.H + parent.Content.Y + b.Dimensions.Margin.Top + b.Dimensions.Border.Top + b.Dimensions.Padding.Top

	}
}

func (b *Box) calculateHeight(parent *Dimensions) {
	switch b.OuterType {
	case OuterBoxType_Block:
		heightSize := b.Style.SpecifiedValues.GetPropAsSize("height")

		if heightSize.IsNone() {
			break
		}
		var height float64
		size := heightSize.Value
		switch size.Kind {
		case cssom.SizeKindKeyword:
		case cssom.SizeKindLP:
			switch size.LP.Unit {
			case "px":
				height = size.LP.Value
			case "%":
				height = parent.Content.H * size.LP.Value / 100
			}
		}

		b.Dimensions.Content.H = height
	}
}

func (b *Box) calculateChildDimensions() {
	for _, child := range b.Children {
		child.calculateDimensions(&b.Dimensions)

		mb := child.Dimensions.MarginBox()
		b.Dimensions.Content.H += mb.H
	}
}

func NewAnonymousBox() *Box {
	return &Box{
		Anonymous: true,
		OuterType: OuterBoxType_Block,
		InnerType: InnerBoxType_Flow,
		Children:  make([]*Box, 0),
	}
}
