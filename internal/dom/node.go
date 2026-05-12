package dom

import (
	"github.com/VisualSource/plex/internal/utils"
)

type Node interface {
	IsNode() uint
	Tag() string
	Namespace() Namespace
	AppendChild(node Node)
	PrependChild(node Node)
	InsertBefore(node Node, ref Node)
	Parent() Node
	// setParent updates a node's parent pointer. Used by AppendChild/PrependChild/
	// InsertBefore to keep parent backrefs consistent when nodes are moved.
	SetParent(Node)
	// removes the element from its parent node. If it has no parent node, calling remove() does nothing.
	Remove()
	// removes a child node from the DOM and returns the removed node.
	RemoveChild(node Node) Node
	Document() *Document
	PreviousSibling() Node
	Children() []Node
}

type ElementNode interface {
	Node
	SetAttribute(key string, value string)
	SetAttributeNS(namespace Namespace, key string, value string)
	GetAttribute(key string) *Attribute
	GetAttributeNS(namespace Namespace, localName string) *Attribute
	SetAttributeNode(node *Attribute)
	HasAttribute(key string) bool
	Attributes() []*Attribute
}

type Attribute struct {
	NamespaceUri Namespace
	Prefix       utils.StringOption
	LocalName    string
	Value        string
}

func (attr Attribute) GetName() string {
	key := attr.LocalName
	if attr.Prefix.IsSome() {
		key = *attr.Prefix.Value + ":" + attr.LocalName
	}
	return key
}

func NewAttribute(namespace Namespace, name string, value string) *Attribute {
	return &Attribute{
		NamespaceUri: namespace,
		Prefix:       utils.None[string](),
		LocalName:    name,
		Value:        value,
	}
}

// https://dom.spec.whatwg.org/#concept-create-element
func NewElement(
	document *Document,
	localName string,
	namespace NamespaceOption,
	prefix utils.StringOption,
	is utils.StringOption,
	synchronousCustomElement bool,
	registry utils.StringOption,
	parent Node) ElementNode {
	ns := utils.ValueOf(namespace.Value, NamespaceHTML)
	if ns == NamespaceHTML {
		switch localName {
		case "template":
			return &TemplateElement{
				document:   document,
				parent:     parent,
				attributes: make([]*Attribute, 0),
			}
		case "selectedcontent":
			return NewSelectedContentElement()
		}

	}

	return &Element{
		document:   document,
		localName:  localName,
		namespace:  ns,
		prefix:     prefix,
		Is:         is,
		parent:     parent,
		attributes: make([]*Attribute, 0),
	}
}

// adoptNode detaches node from its current parent (if it has one that isn't
// newParent) so it can be re-parented under newParent. The old parent's
// children list is updated; the node's parent pointer is left for the caller
// to update via setParent.
func adoptNode(node Node, newParent Node) {
	if node == nil {
		return
	}
	current := node.Parent()
	if current != nil && current != newParent {
		current.RemoveChild(node)
	}
}
