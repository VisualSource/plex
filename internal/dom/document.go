package dom

type Document struct {
	children []Node
}

func (d *Document) AppendChild(node Node) {
	d.children = append(d.children, node)
}
