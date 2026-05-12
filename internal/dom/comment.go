package dom

type Comment struct {
	Data     string
	document *Document
	parent   Node
}

func NewComment(document *Document, parent Node, data string) *Comment {
	return &Comment{Data: data, document: document, parent: parent}
}

func (c Comment) Parent() Node {
	return c.parent
}
func (c *Comment) SetParent(parent Node) {
	c.parent = parent
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
func (c *Comment) PrependChild(node Node) {
	panic("Comment node does not support this method")
}
func (c *Comment) InsertBefore(node Node, ref Node) {
	panic("Comment node does not support this method")
}
func (c Comment) Document() *Document {
	return c.document
}
func (c Comment) PreviousSibling() Node {
	return nil
}
func (c Comment) Children() []Node {
	return nil
}
func (c Comment) RemoveChild(Node) Node {
	return nil
}
func (c *Comment) Remove() {
	if c.parent != nil {
		c.parent.RemoveChild(c)
	}
}
