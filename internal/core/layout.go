package plex

import "github.com/moznion/go-optional"

type DisplayType = uint8

const (
	DisplayNone        DisplayType = 0
	DisplayBlock       DisplayType = 1
	DisplayInline      DisplayType = 2
	DisplayInlineBlock DisplayType = 3
	DisplayFlex        DisplayType = 4
)

type Rect struct {
	x      float64
	y      float64
	width  float64
	height float64
}

func (r *Rect) expandedBy(edge EdgeSizes) {}

type EdgeSizes struct {
	left   float64
	right  float64
	top    float64
	bottom float64
}

type Dimensions struct {
	content Rect
	padding EdgeSizes
	border  EdgeSizes
	margin  EdgeSizes
}

func (d *Dimensions) paddingBox() {}
func (d *Dimensions) borderBox()  {}
func (d *Dimensions) marginBox()  {}

type ContainerDisplay = uint8

const (
	ContainerDisplayBlock          ContainerDisplay = 0
	ContainerDisplayInline         ContainerDisplay = 1
	ContainerDisplayAnonymousBlock ContainerDisplay = 2
)

type LayoutBox struct {
	dimensions Dimensions
	container  ContainerDisplay
	node       optional.Option[StyledNode]
	children   []LayoutBox
}

func (l *LayoutBox) GetInlineContainer() LayoutBox {
	switch l.container {
	case ContainerDisplayInline:
		fallthrough
	case ContainerDisplayAnonymousBlock:
		return *l
	case ContainerDisplayBlock:

		if len(l.children) <= 0 {
			l.children = append(l.children, LayoutBox{
				container: ContainerDisplayAnonymousBlock,
			})
		}

		last := l.children[len(l.children)-1]

		return last
	}

	panic("Should not be here")
}

func (l *LayoutBox) GetLayout(containing Dimensions) {
	switch l.container {
	case ContainerDisplayBlock:
		l.layoutBlock(containing)
	case ContainerDisplayInline:
	case ContainerDisplayAnonymousBlock:
	}
}

func (l *LayoutBox) layoutBlock(containing Dimensions) {
	l.calculateBlockWidth(containing)
	l.calculateBlockPosition(containing)
	l.layoutBlockChildren()
	l.calculateBlockHeight()
}

func (l *LayoutBox) calculateBlockWidth(containing Dimensions) {
	if l.node.IsNone() {
		return
	}

	style := l.node.Unwrap()
	var keywordAuto CssValue = "auto"
	width := style.GetProp("width").Or(optional.Some(keywordAuto)).Unwrap()

}
func (l *LayoutBox) calculateBlockPosition(containing Dimensions) {}
func (l *LayoutBox) layoutBlockChildren()                         {}
func (l *LayoutBox) calculateBlockHeight()                        {}

func getContainerDisplay(display DisplayType) ContainerDisplay {
	switch display {
	case DisplayBlock:
		return ContainerDisplayBlock
	case DisplayInline:
		return ContainerDisplayInline
	default:
		return ContainerDisplayAnonymousBlock
	}
}

func BuildLayoutTree(node StyledNode) LayoutBox {

	root := LayoutBox{
		container: getContainerDisplay(node.GetDisplay()),
		node:      optional.Some(node),
	}

	for _, child := range node.children {
		switch child.GetDisplay() {
		case DisplayBlock:
			root.children = append(root.children, BuildLayoutTree(child))
		case DisplayInline:
			inline := root.GetInlineContainer()
			inline.children = append(inline.children, BuildLayoutTree(child))
		}
	}

	return root
}
