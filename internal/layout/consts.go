package layout

import (
	"github.com/VisualSource/plex/internal/css/cssom"
	"github.com/VisualSource/plex/internal/layout/styletree"
	"github.com/VisualSource/plex/internal/utils"
	"github.com/Zyko0/go-sdl3/sdl"
)

type Edge struct {
	Top    float32
	Left   float32
	Right  float32
	Bottom float32
}

type Dimensions struct {
	Content sdl.FRect
	Padding Edge
	border  Edge
	margin  Edge
}

func (d *Dimensions) PaddingBox() sdl.FRect {
	return sdl.FRect{
		H: d.Content.H + d.Padding.Top + d.Padding.Bottom,
		W: d.Content.W + d.Padding.Left + d.Padding.Right,
		X: d.Content.X - d.Padding.Left,
		Y: d.Content.Y - d.Padding.Top,
	}
}
func (d *Dimensions) BorderBox() sdl.FRect {
	pb := d.PaddingBox()

	pb.X -= d.border.Left
	pb.Y -= d.border.Top

	pb.W += d.border.Left + d.border.Right
	pb.H += d.border.Top + d.border.Bottom

	return pb
}
func (d *Dimensions) MarginBox() sdl.FRect {
	bb := d.BorderBox()

	bb.X -= d.margin.Left
	bb.Y -= d.margin.Top

	bb.W += d.margin.Left + d.margin.Right
	bb.H += d.margin.Top + d.margin.Bottom

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
	Node       *styletree.StyledNode
	Children   []Box
}

func (b *Box) getInlineContainer() *Box {
	switch b.OuterType {
	case OuterBoxType_Block:
		if len(b.Children) > 1 && b.Children[len(b.Children)-1].Anonymous {
			return &b.Children[len(b.Children)-1]
		} else {
			box := NewAnonymousBox()

			b.Children = append(b.Children, box)

			return &box
		}
	default:
		return b
	}
}

func (b *Box) calculateDimensions(dimensions *Dimensions) {
	b.calculateWidth(dimensions)
	b.calculatePosition(dimensions)
	b.calculateChildDimensions()

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

func (b *Box) calculateWidth(parent *Dimensions) {
	switch b.OuterType {
	case OuterBoxType_Block:
		sizes := []utils.Option[cssom.Size]{
			b.Node.SpecifiedValues.GetPropAsSize("margin-left"),
			b.Node.SpecifiedValues.GetPropAsSize("margin-right"),

			b.Node.SpecifiedValues.GetPropAsSize("border-left-width"),
			b.Node.SpecifiedValues.GetPropAsSize("border-right-width"),

			b.Node.SpecifiedValues.GetPropAsSize("padding-left"),
			b.Node.SpecifiedValues.GetPropAsSize("padding-right"),

			b.Node.SpecifiedValues.GetPropAsSize("width"),
		}

		var total float32

		values := make([]float32, 7)
		autodFields := make([]bool, 7)
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
					total += float32(size.LP.Value)

					values[i] = float32(size.LP.Value)
				case "%":
					v := parent.Content.W * float32(size.LP.Value/100)
					total += v
					values[i] = v
				}
			}
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
		b.Dimensions.border.Left = values[field_BorderLeft]
		b.Dimensions.border.Right = values[field_BorderRight]
		b.Dimensions.margin.Left = values[field_MarginLeft]
		b.Dimensions.margin.Right = values[field_MarginRight]
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
			b.Node.SpecifiedValues.GetPropAsSize("margin-top"),
			b.Node.SpecifiedValues.GetPropAsSize("margin-bottom"),

			b.Node.SpecifiedValues.GetPropAsSize("border-top-width"),
			b.Node.SpecifiedValues.GetPropAsSize("border-bottom-width"),

			b.Node.SpecifiedValues.GetPropAsSize("padding-top"),
			b.Node.SpecifiedValues.GetPropAsSize("padding-bottom"),
		}

		values := make([]float32, 6)
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

					values[i] = float32(size.LP.Value)
				case "%":
					v := parent.Content.H * float32(size.LP.Value/100)
					//total += v
					values[i] = v
				}
			}
		}

		b.Dimensions.margin.Top = values[field_MarginTop]
		b.Dimensions.margin.Bottom = values[field_MarginBottom]

		b.Dimensions.border.Top = values[field_BorderTop]
		b.Dimensions.border.Bottom = values[field_BorderBottom]

		b.Dimensions.Padding.Top = values[field_PaddingTop]
		b.Dimensions.Padding.Bottom = values[field_PaddingBottom]

		b.Dimensions.Content.X += parent.Content.X + b.Dimensions.margin.Left + b.Dimensions.border.Left + b.Dimensions.Padding.Left
		b.Dimensions.Content.Y = parent.Content.H + parent.Content.Y + b.Dimensions.margin.Top + b.Dimensions.border.Top + b.Dimensions.Padding.Top

	}
}

func (b *Box) calculateHeight(parent *Dimensions) {
	switch b.OuterType {
	case OuterBoxType_Block:
		heightSize := b.Node.SpecifiedValues.GetPropAsSize("height")

		if heightSize.IsNone() {
			break
		}
		var height float32
		size := heightSize.Value
		switch size.Kind {
		case cssom.SizeKindKeyword:
		case cssom.SizeKindLP:
			switch size.LP.Unit {
			case "px":
				height = float32(size.LP.Value)
			case "%":
				height = parent.Content.H * float32(size.LP.Value/100)
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

func NewAnonymousBox() Box {
	return Box{
		Anonymous: true,
		OuterType: OuterBoxType_Block,
		InnerType: InnerBoxType_Flow,
		Children:  make([]Box, 0),
	}
}
