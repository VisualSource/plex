package dom

type DocumentType struct {
	Name     string
	PublicId string
	SystemId string
	document *Document
	parent   Node
}

func (d DocumentType) Parent() Node {
	return d.parent
}
func (d *DocumentType) SetParent(parent Node) {
	d.parent = parent
}
func (d DocumentType) IsNode() uint {
	return 1
}
func (d DocumentType) Tag() string {
	return "#doctype"
}
func (d DocumentType) Namespace() Namespace {
	return NamespaceHTML
}
func (d *DocumentType) AppendChild(node Node) {
	panic("DocumentType node does not support this method")
}
func (d *DocumentType) PrependChild(node Node) {
	panic("DocumentType node does not support this method")
}
func (d *DocumentType) InsertBefore(node Node, ref Node) {
	panic("DocumentType node does not support this method")
}
func (d DocumentType) Document() *Document {
	return d.document
}
func (d DocumentType) PreviousSibling() Node {
	return nil
}
func (d DocumentType) Children() []Node {
	return nil
}
func (d DocumentType) Remove() {}
func (d DocumentType) RemoveChild(Node) Node {
	return nil
}

func NewDocumentType(name string, publicId string, systemId string) *DocumentType {
	return &DocumentType{Name: name, PublicId: publicId, SystemId: systemId}
}
