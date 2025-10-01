package html

import "slices"

type DocumentType struct {
	name     string
	systemId string
	publicId string
}

func NewDocumentType(name string, systemId string, publicId string) DocumentType {
	return DocumentType{
		name:     name,
		systemId: systemId,
		publicId: publicId,
	}
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
