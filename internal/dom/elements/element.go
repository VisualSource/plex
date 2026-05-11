package elements

import (
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/utils"
)

type Element struct {
	children   []dom.Node
	localName  string
	namespace  dom.Namespace
	Is         utils.StringOption
	document   *dom.Document
	attributes []*dom.Attribute
	prefix     utils.StringOption
	parent     dom.Node
	shadowRoot *ShadowRoot
}

func (e *Element) IsShadowHost() bool         { return e.shadowRoot != nil }
func (e *Element) GetShadowRoot() *ShadowRoot { return e.shadowRoot }

func (e *Element) AttachShadow(doc *dom.Document, mode, slotAssignment string, clonable, serializable, delegatesFocus, keepRegistryNull bool) (*ShadowRoot, error) {
	shadow := NewShadowRoot(doc, e, mode, slotAssignment, clonable, serializable, delegatesFocus, keepRegistryNull)
	e.shadowRoot = shadow
	return shadow, nil
}

func (e Element) Parent() dom.Node {
	return e.parent
}
func (e *Element) SetParent(node dom.Node) {
	e.parent = node
}
func (e Element) IsNode() uint {
	return 4
}
func (e Element) Tag() string {
	return e.localName
}
func (e Element) Namespace() dom.Namespace {
	return e.namespace
}
func (e *Element) AppendChild(node dom.Node) {
	adoptNode(node, e)
	node.SetParent(e)
	e.children = append(e.children, node)
}
func (e *Element) PrependChild(node dom.Node) {
	adoptNode(node, e)
	node.SetParent(e)
	e.children = slices.Insert(e.children, 0, node)
}
func (e *Element) InsertBefore(node dom.Node, ref dom.Node) {
	adoptNode(node, e)
	node.SetParent(e)
	idx := slices.Index(e.children, ref)
	if idx == -1 {
		e.children = append(e.children, node)
	} else {
		e.children = slices.Insert(e.children, idx, node)
	}
}
func (e Element) Document() *dom.Document {
	return e.document
}
func (e Element) PreviousSibling() dom.Node {
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
func (e Element) Children() []dom.Node {
	return e.children
}
func (e *Element) SetAttributeNode(node *dom.Attribute) {
	e.attributes = append(e.attributes, node)
}
func (e *Element) SetAttribute(key string, value string) {
	e.attributes = append(e.attributes, dom.NewAttribute(dom.NamespaceHTML, key, value))
}
func (e *Element) SetAttributeNS(namespace dom.Namespace, key string, value string) {
	e.attributes = append(e.attributes, dom.NewAttribute(namespace, key, value))
}
func (e Element) GetAttribute(key string) *dom.Attribute {
	name := strings.ToLower(key)

	for _, attr := range e.attributes {
		qualName := attr.LocalName
		if attr.Prefix.IsSome() {
			qualName = *attr.Prefix.Value + ":" + attr.LocalName
		}

		if qualName == name {
			return attr
		}
	}

	return nil
}
func (e Element) GetAttributeNS(namespace dom.Namespace, localname string) *dom.Attribute {
	for _, attr := range e.attributes {
		if attr.NamespaceUri == namespace && attr.LocalName == localname {
			return attr
		}
	}

	return nil
}
func (e Element) HasAttribute(key string) bool {
	attr := e.GetAttribute(key)
	return attr != nil
}
func (e Element) Attributes() []*dom.Attribute {
	return e.attributes
}
func (e *Element) Remove() {
	if e.parent != nil {
		e.parent.RemoveChild(e)
	}
}
func (e *Element) RemoveChild(node dom.Node) dom.Node {
	idx := slices.Index(e.children, node)
	if idx == -1 {
		return nil
	}

	removed := e.children[idx]
	e.children = slices.Delete(e.children, idx, idx+1)

	return removed
}
