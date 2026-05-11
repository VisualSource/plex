package dom

import "slices"

type Document struct {
	children                    []Node
	QuirksMode                  QuirksMode
	ParserNoChangeMode          bool
	AllowDeclarativeShadowRoots bool
}

func (d Document) Parent() Node {
	return nil
}
func (d *Document) SetParent(parent Node) {}
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
func (d *Document) InsertBefore(node Node, ref Node) {
	idx := slices.Index(d.children, ref)
	if idx == -1 {
		d.children = append(d.children, node)
	} else {
		d.children = slices.Insert(d.children, idx, node)
	}
}
func (d Document) Remove() {}
func (d *Document) RemoveChild(node Node) Node {
	idx := slices.Index(d.children, node)
	if idx == -1 {
		return nil
	}

	removed := d.children[idx]
	d.children = slices.Delete(d.children, idx, idx+1)
	return removed
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
		ParserNoChangeMode:          false,
		AllowDeclarativeShadowRoots: true,
		QuirksMode:                  QuirksMode_NoQuirks,
		children:                    make([]Node, 0),
	}
}
