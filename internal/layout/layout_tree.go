package layout

import (
	"github.com/VisualSource/plex/internal/layout/styletree"
)

func NewLayoutTree(node *styletree.StyledNode) Box {

	dim := &Dimensions{}

	box := buildLayoutTree(node)
	box.calculateDimensions(dim)

	return box
}

func buildLayoutTree(node *styletree.StyledNode) Box {

	outer, inner := getDisplayValue(node.SpecifiedValues.GetPropAsDisplay("display"))

	box := Box{
		Node:      node,
		OuterType: outer,
		InnerType: inner,
		Children:  make([]Box, 0),
	}

	for _, child := range node.Children {
		outer, _ := getDisplayValue(child.SpecifiedValues.GetPropAsDisplay("display"))

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
