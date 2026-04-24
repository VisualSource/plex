package dom

type Node interface {
	IsNode() uint
	Tag() string
	Namespace() Namespace
	AppendChild(node Node)
}

type DocumentType struct {
	Name     string
	PublicId string
	SystemId string
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
func NewDocumentType(name string, publicId string, systemId string) *DocumentType {
	return &DocumentType{Name: name, PublicId: publicId, SystemId: systemId}
}

type Text struct {
	Data string
}

func (t Text) IsNode() uint {
	return 2
}
func (t Text) Tag() string {
	return "#text"
}
func (t Text) Namespace() Namespace {
	return NamespaceHTML
}
func (t *Text) AppendChild(node Node) {
	panic("Text node does not support this method")
}

func NewTextNode(data string) *Text {
	return &Text{Data: data}
}

type Comment struct {
	Data string
}

func (c Comment) IsNode() uint {
	return 3
}
func (c Comment) Tag() string {
	return "#comment"
}
func (c Comment) Namespace() Namespace {
	return NamespaceHTML
}
func (c *Comment) AppendChild(node Node) {
	panic("Comment node does not support this method")
}
func NewComment(data string) *Comment {
	return &Comment{Data: data}
}

type Element struct {
	children  []Node
	localName string
	namespace Namespace
}

func (e Element) IsNode() uint {
	return 4
}
func (e Element) Tag() string {
	return e.localName
}
func (e Element) Namespace() Namespace {
	return e.namespace
}
func (e *Element) AppendChild(node Node) {
	e.children = append(e.children, node)
}
func NewElement(tagName string, namespace Namespace) *Element {
	return &Element{
		localName: tagName,
		namespace: namespace,
	}
}
