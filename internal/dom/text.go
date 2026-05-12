package dom

type Text struct {
	Data     string
	document *Document
	parent   Node
}

func NewTextNode(document *Document, data string, parent Node) *Text {
	return &Text{Data: data, document: document, parent: parent}
}

func (t Text) Parent() Node {
	return t.parent
}
func (t *Text) SetParent(node Node) {
	t.parent = node
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
func (t *Text) PrependChild(node Node) {
	panic("Text node does not support this method")
}
func (t *Text) InsertBefore(node Node, ref Node) {
	panic("Text node does not support this method")
}
func (t Text) Document() *Document {
	return t.document
}
func (t *Text) Remove() {
	if t.parent != nil {
		t.parent.RemoveChild(t)
	}
}
func (t *Text) RemoveChild(Node) Node {
	return nil
}
func (t Text) PreviousSibling() Node {
	return nil
}
func (t Text) Children() []Node {
	return nil
}
