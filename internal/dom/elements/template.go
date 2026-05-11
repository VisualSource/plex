package elements

import (
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/dom"
)

type TemplateElement struct {
	templateContents []dom.Node
	shadowContents   *ShadowRoot
	document         *dom.Document
	parent           dom.Node
	attributes       []*dom.Attribute
}

func (e *TemplateElement) SetTemplateContents(shadow *ShadowRoot) {
	e.shadowContents = shadow
}

func (e TemplateElement) Document() *dom.Document {
	return e.document
}

func (e TemplateElement) Parent() dom.Node {
	return e.parent
}
func (e TemplateElement) SetParent(node dom.Node) {
	e.parent = node
}
func (e TemplateElement) IsNode() uint {
	return 4
}
func (e TemplateElement) Tag() string {
	return "template"
}
func (e TemplateElement) Namespace() dom.Namespace {
	return dom.NamespaceHTML
}
func (e *TemplateElement) AppendChild(node dom.Node) {
	if e.shadowContents != nil {
		e.shadowContents.AppendChild(node)
		return
	}
	adoptNode(node, e)
	node.SetParent(e)
	e.templateContents = append(e.templateContents, node)
}
func (e *TemplateElement) PrependChild(node dom.Node) {
	if e.shadowContents != nil {
		e.shadowContents.PrependChild(node)
		return
	}
	adoptNode(node, e)
	node.SetParent(e)
	e.templateContents = slices.Insert(e.templateContents, 0, node)
}
func (e *TemplateElement) InsertBefore(node dom.Node, ref dom.Node) {
	if e.shadowContents != nil {
		e.shadowContents.InsertBefore(node, ref)
		return
	}
	adoptNode(node, e)
	node.SetParent(e)
	idx := slices.Index(e.templateContents, ref)
	if idx == -1 {
		e.templateContents = append(e.templateContents, node)
	} else {
		e.templateContents = slices.Insert(e.templateContents, idx, node)
	}
}
func (e TemplateElement) PreviousSibling() dom.Node {
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
func (e TemplateElement) Children() []dom.Node {
	return e.templateContents
}
func (e *TemplateElement) SetAttributeNode(node *dom.Attribute) {
	e.attributes = append(e.attributes, node)
}
func (e *TemplateElement) SetAttribute(key string, value string) {
	e.attributes = append(e.attributes, dom.NewAttribute(dom.NamespaceHTML, key, value))
}
func (e *TemplateElement) SetAttributeNS(namespace dom.Namespace, key string, value string) {
	e.attributes = append(e.attributes, dom.NewAttribute(namespace, key, value))
}
func (e TemplateElement) GetAttribute(key string) *dom.Attribute {
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
func (e TemplateElement) GetAttributeNS(namespace dom.Namespace, localname string) *dom.Attribute {
	for _, attr := range e.attributes {
		if attr.NamespaceUri == namespace && attr.LocalName == localname {
			return attr
		}
	}

	return nil
}
func (e TemplateElement) HasAttribute(key string) bool {
	attr := e.GetAttribute(key)
	return attr != nil
}
func (e TemplateElement) Attributes() []*dom.Attribute {
	return e.attributes
}
func (e *TemplateElement) Remove() {
	if e.parent != nil {
		e.parent.RemoveChild(e)
	}
}
func (e *TemplateElement) RemoveChild(node dom.Node) dom.Node {
	idx := slices.Index(e.templateContents, node)
	if idx != -1 {
		return nil
	}

	removed := e.templateContents[idx]
	e.templateContents = slices.Delete(e.templateContents, idx, idx+1)

	return removed
}
