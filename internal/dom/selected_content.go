package dom

import "github.com/VisualSource/plex/internal/utils"

type SelectedContentElement struct {
	parent     Node
	namespace  Namespace
	document   *Document
	attributes []*Attribute
}

func (e SelectedContentElement) IsNode() uint {
	return 4
}
func (e SelectedContentElement) Tag() string {
	return "selectedcontent"
}
func (e SelectedContentElement) Namespace() Namespace {
	return e.namespace
}
func (e *SelectedContentElement) AppendChild(node Node)            {}
func (e *SelectedContentElement) PrependChild(node Node)           {}
func (e *SelectedContentElement) InsertBefore(node Node, ref Node) {}
func (e SelectedContentElement) Parent() Node {
	return e.parent
}
func (e *SelectedContentElement) SetParent(node Node) {
	e.parent = node
}
func (e *SelectedContentElement) Remove() {
	if e.parent != nil {
		e.parent.RemoveChild(e)
	}
}
func (e *SelectedContentElement) RemoveChild(node Node) Node {
	return nil
}
func (e SelectedContentElement) Document() *Document {
	return e.document
}
func (e SelectedContentElement) PreviousSibling() Node {
	return previousSibling(&e)
}
func (e SelectedContentElement) Children() []Node {
	parent := e.Parent()
	if parent == nil {
		return nil
	}
	if parent.Tag() != "button" {
		return nil
	}
	parentParent := parent.Parent()
	if parentParent == nil {
		return nil
	}
	if parentParent.Tag() != "select" {
		return nil
	}

	children := parentParent.Children()

	firstIdx := -1
	for i := 0; i < len(children); i++ {
		child := children[i]
		if child.Tag() != "option" {
			continue
		}

		if firstIdx == -1 {
			firstIdx = i
		}

		if el, ok := child.(ElementNode); ok {
			if el.HasAttribute("selected") {
				return el.Children()
			}
		}
	}

	if firstIdx != -1 {
		return children[firstIdx].Children()
	}

	return nil
}

func (e *SelectedContentElement) SetAttribute(key string, value string) {
	e.attributes = append(e.attributes, NewAttribute(NamespaceHTML, key, value))
}
func (e *SelectedContentElement) SetAttributeNS(namespace Namespace, key string, value string) {
	e.attributes = append(e.attributes, NewAttribute(namespace, key, value))
}
func (e SelectedContentElement) GetAttribute(key string) *Attribute {
	return nil
}
func (e SelectedContentElement) GetAttributeNS(namespace Namespace, key string) *Attribute {
	return nil
}
func (e *SelectedContentElement) SetAttributeNode(node *Attribute) {
	e.attributes = append(e.attributes, node)
}
func (e SelectedContentElement) HasAttribute(key string) bool {
	return false
}
func (e SelectedContentElement) Attributes() []*Attribute {
	return nil
}

func NewSelectedContentElement() *SelectedContentElement {
	return &SelectedContentElement{}
}

func (e SelectedContentElement) Classes() utils.StringOption {
	attr := e.GetAttribute("class")
	if attr != nil {
		return utils.Some(attr.Value)
	}

	return utils.None[string]()
}
func (e SelectedContentElement) Id() utils.StringOption {
	attr := e.GetAttribute("id")
	if attr != nil {
		return utils.Some(attr.Value)
	}

	return utils.None[string]()
}
