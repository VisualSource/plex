package html

import "slices"

type Node interface {
	GetNodeName() string
	GetNodeType() uint
}

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
	return 1
}

func (c *HTMLElement) GetNodeName() string {
	return c.name
}

type CommentNode struct {
	data string
}

func (c *CommentNode) GetNodeType() uint {
	return 8
}

func (c *CommentNode) GetNodeName() string {
	return "#comment"
}

func NewCommentNode(data string) *CommentNode {
	return &CommentNode{
		data: data,
	}
}

type DocumentType struct {
	name     string
	systemId string
	publicId string
}

type Document struct {
	title    string
	head     Node
	body     Node
	children []Node
	doctype  DocumentType
}

func (d *Document) Prepend(node Node) {
	d.children = slices.Insert(d.children, 0, node)
}
func (d *Document) Append(node Node) {
	d.children = append(d.children, node)
}
