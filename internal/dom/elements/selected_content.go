package elements

import (
	"slices"

	"github.com/VisualSource/plex/internal/dom"
)

type SelectedContentElement struct {
	parent     dom.Node
	namespace  dom.Namespace
	document   *dom.Document
	attributes []*dom.Attribute
}

func (e SelectedContentElement) IsNode() uint {
	return 4
}
func (e SelectedContentElement) Tag() string {
	return "selectedcontent"
}
func (e SelectedContentElement) Namespace() dom.Namespace {
	return e.namespace
}
func (e *SelectedContentElement) AppendChild(node dom.Node)                {}
func (e *SelectedContentElement) PrependChild(node dom.Node)               {}
func (e *SelectedContentElement) InsertBefore(node dom.Node, ref dom.Node) {}
func (e SelectedContentElement) Parent() dom.Node {
	return e.parent
}
func (e *SelectedContentElement) SetParent(node dom.Node) {
	e.parent = node
}
func (e *SelectedContentElement) Remove() {
	if e.parent != nil {
		e.parent.RemoveChild(e)
	}
}
func (e *SelectedContentElement) RemoveChild(node dom.Node) dom.Node {
	return nil
}
func (e SelectedContentElement) Document() *dom.Document {
	return e.document
}
func (e SelectedContentElement) PreviousSibling() dom.Node {
	parent := e.Parent()

	if parent == nil {
		return nil
	}

	children := parent.Children()
	if children == nil {
		return nil
	}

	idx := slices.IndexFunc(children, func(node dom.Node) bool {
		return node == &e
	})

	if idx == -1 || idx-1 < 0 {
		return nil
	}

	return children[idx-1]
}
func (e SelectedContentElement) Children() []dom.Node {
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

		if el, ok := child.(dom.ElementNode); ok {
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
	e.attributes = append(e.attributes, dom.NewAttribute(dom.NamespaceHTML, key, value))
}
func (e *SelectedContentElement) SetAttributeNS(namespace dom.Namespace, key string, value string) {
	e.attributes = append(e.attributes, dom.NewAttribute(namespace, key, value))
}
func (e SelectedContentElement) GetAttribute(key string) *dom.Attribute {
	return nil
}
func (e SelectedContentElement) GetAttributeNS(namespace dom.Namespace, key string) *dom.Attribute {
	return nil
}
func (e *SelectedContentElement) SetAttributeNode(node *dom.Attribute) {
	e.attributes = append(e.attributes, node)
}
func (e SelectedContentElement) HasAttribute(key string) bool {
	return false
}
func (e SelectedContentElement) Attributes() []*dom.Attribute {
	return nil
}

func NewSelectedContentElement() *SelectedContentElement {
	return &SelectedContentElement{}
}
