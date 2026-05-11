package elements

import (
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/utils"
)

// https://dom.spec.whatwg.org/#concept-create-element
func NewElement(
	document *dom.Document,
	localName string,
	namespace dom.NamespaceOption,
	prefix utils.StringOption,
	is utils.StringOption,
	synchronusCustomElement bool,
	registry utils.StringOption,
	parent dom.Node) dom.ElementNode {
	ns := utils.ValueOf(namespace.Value, dom.NamespaceHTML)
	if ns == dom.NamespaceHTML {
		switch localName {
		case "template":
			return &TemplateElement{
				document:   document,
				parent:     parent,
				attributes: make([]*dom.Attribute, 0),
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
		attributes: make([]*dom.Attribute, 0),
	}
}

// adoptNode detaches node from its current parent (if it has one that isn't
// newParent) so it can be re-parented under newParent. The old parent's
// children list is updated; the node's parent pointer is left for the caller
// to update via setParent.
func adoptNode(node dom.Node, newParent dom.Node) {
	if node == nil {
		return
	}
	current := node.Parent()
	if current != nil && current != newParent {
		current.RemoveChild(node)
	}
}
