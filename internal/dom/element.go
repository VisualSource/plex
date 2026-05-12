package dom

import (
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/utils"
)

type Element struct {
	children   []Node
	localName  string
	namespace  Namespace
	Is         utils.StringOption
	document   *Document
	attributes []*Attribute
	prefix     utils.StringOption
	parent     Node
	shadowRoot *ShadowRoot
}

func (e *Element) IsShadowHost() bool         { return e.shadowRoot != nil }
func (e *Element) GetShadowRoot() *ShadowRoot { return e.shadowRoot }

func (e *Element) AttachShadow(doc *Document, mode ShadowRootMode, slotAssignment ShadowRootSlotAssignment, clonable, serializable, delegatesFocus, keepRegistryNull bool) (*ShadowRoot, error) {
	shadow := NewShadowRoot(doc, e, mode, slotAssignment, clonable, serializable, delegatesFocus, keepRegistryNull)
	e.shadowRoot = shadow
	return shadow, nil
}

func (e Element) Parent() Node {
	return e.parent
}
func (e *Element) SetParent(node Node) {
	e.parent = node
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
	adoptNode(node, e)
	node.SetParent(e)
	e.children = append(e.children, node)
}
func (e *Element) PrependChild(node Node) {
	adoptNode(node, e)
	node.SetParent(e)
	e.children = slices.Insert(e.children, 0, node)
}
func (e *Element) InsertBefore(node Node, ref Node) {
	adoptNode(node, e)
	node.SetParent(e)
	idx := slices.Index(e.children, ref)
	if idx == -1 {
		e.children = append(e.children, node)
	} else {
		e.children = slices.Insert(e.children, idx, node)
	}
}
func (e Element) Document() *Document {
	return e.document
}
func (e Element) PreviousSibling() Node {
	return previousSibling(&e)
}
func (e Element) Children() []Node {
	return e.children
}
func (e *Element) SetAttributeNode(node *Attribute) {
	e.attributes = append(e.attributes, node)
}
func (e *Element) SetAttribute(key string, value string) {
	e.attributes = append(e.attributes, NewAttribute(NamespaceHTML, key, value))
}
func (e *Element) SetAttributeNS(namespace Namespace, key string, value string) {
	e.attributes = append(e.attributes, NewAttribute(namespace, key, value))
}
func (e Element) GetAttribute(key string) *Attribute {
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
func (e Element) GetAttributeNS(namespace Namespace, localname string) *Attribute {
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
func (e Element) Attributes() []*Attribute {
	return e.attributes
}
func (e *Element) Remove() {
	if e.parent != nil {
		e.parent.RemoveChild(e)
	}
}
func (e *Element) RemoveChild(node Node) Node {
	idx := slices.Index(e.children, node)
	if idx == -1 {
		return nil
	}

	removed := e.children[idx]
	e.children = slices.Delete(e.children, idx, idx+1)

	return removed
}
