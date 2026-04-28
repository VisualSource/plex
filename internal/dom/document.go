package dom

import "slices"

type Document struct {
	children           []Node
	QuirksMode         QuirksMode
	ParserNoChangeMode bool
}

func (d Document) Parent() Node {
	return nil
}
func (d Document) IsNode() uint {
	return 0
}
func (d Document) Tag() string {
	return "#document"
}
func (d Document) Namespace() Namespace {
	return NamespaceHTML
}
func (d Document) PreviousSibling() Node {
	return nil
}
func (d *Document) AppendChild(node Node) {
	d.children = append(d.children, node)
}
func (d *Document) PrependChild(node Node) {
	d.children = slices.Insert(d.children, 0, node)
}
func (d Document) Children() []Node {
	return d.children
}
func (d Document) IsIframeSrcDoc() bool {
	return false
}
func (d Document) Document() *Document {
	return &d
}

func NewDocument() *Document {
	return &Document{
		ParserNoChangeMode: false,
		QuirksMode:         QuirksMode_NoQuirks,
		children:           make([]Node, 0),
	}
}
