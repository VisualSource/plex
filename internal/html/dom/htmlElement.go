package html

import "slices"

type HTMLElement struct {
	children []Node
	name     string
}

func NewHTMLElement(tag string) *HTMLElement {
	return &HTMLElement{
		name:     tag,
		children: make([]Node, 0),
	}
}

func (d *HTMLElement) Prepend(node Node) {
	d.children = slices.Insert(d.children, 0, node)
}
func (d *HTMLElement) Append(node Node) {
	d.children = append(d.children, node)
}

func (c *HTMLElement) GetNodeType() uint {
	return Node_Element
}

func (c *HTMLElement) GetNodeName() string {
	return c.name
}
