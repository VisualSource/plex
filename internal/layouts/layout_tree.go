package layouts

import (
	"github.com/VisualSource/plex/internal/css/cssom"
	"github.com/VisualSource/plex/internal/layouts/styletree"
)

func NewLayoutTree(node *styletree.StyledNode, width, height float64) *Box {

	dim := &Dimensions{
		Content: Rect{
			H: 0, //height, //TODO: need stacking context or something
			W: width,
		},
	}

	box := buildLayoutTree(node)
	box.calculateDimensions(dim)

	return box
}

func buildLayoutTree(node *styletree.StyledNode) *Box {

	outer, inner := getDisplayValue(styletree.GetProp[cssom.Display](node.SpecifiedValues, "display"))

	box := &Box{
		Style:     node,
		OuterType: outer,
		InnerType: inner,
		Children:  make([]*Box, 0),
	}

	for _, child := range node.Children {
		outer, _ := getDisplayValue(styletree.GetProp[cssom.Display](child.SpecifiedValues, "display"))

		switch outer {
		case OuterBoxType_Block:
			b := buildLayoutTree(child)

			box.Children = append(box.Children, b)
		case OuterBoxType_Inline:
			container := box.getInlineContainer()

			b := buildLayoutTree(child)

			container.Children = append(container.Children, b)
		}
	}

	return box
}
