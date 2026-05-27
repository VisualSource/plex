package dom

import (
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/utils"
)

type TemplateElement struct {
	templateContents []Node
	shadowContents   *ShadowRoot
	document         *Document
	parent           Node
	attributes       []*Attribute
}

func (e *TemplateElement) SetTemplateContents(shadow *ShadowRoot) {
	e.shadowContents = shadow
}

func (e TemplateElement) Document() *Document {
	return e.document
}

func (e TemplateElement) Parent() Node {
	return e.parent
}
func (e TemplateElement) SetParent(node Node) {
	e.parent = node
}
func (e TemplateElement) IsNode() uint {
	return 4
}
func (e TemplateElement) Tag() string {
	return "template"
}
func (e TemplateElement) Namespace() Namespace {
	return NamespaceHTML
}
func (e *TemplateElement) AppendChild(node Node) {
	if e.shadowContents != nil {
		e.shadowContents.AppendChild(node)
		return
	}
	adoptNode(node, e)
	node.SetParent(e)
	e.templateContents = append(e.templateContents, node)
}
func (e *TemplateElement) PrependChild(node Node) {
	if e.shadowContents != nil {
		e.shadowContents.PrependChild(node)
		return
	}
	adoptNode(node, e)
	node.SetParent(e)
	e.templateContents = slices.Insert(e.templateContents, 0, node)
}
func (e *TemplateElement) InsertBefore(node Node, ref Node) {
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
func (e TemplateElement) PreviousSibling() Node {
	return previousSibling(&e)
}
func (e TemplateElement) Children() []Node {
	return e.templateContents
}
func (e *TemplateElement) SetAttributeNode(node *Attribute) {
	e.attributes = append(e.attributes, node)
}
func (e *TemplateElement) SetAttribute(key string, value string) {
	e.attributes = append(e.attributes, NewAttribute(NamespaceHTML, key, value))
}
func (e *TemplateElement) SetAttributeNS(namespace Namespace, key string, value string) {
	e.attributes = append(e.attributes, NewAttribute(namespace, key, value))
}
func (e TemplateElement) GetAttribute(key string) *Attribute {
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
func (e TemplateElement) GetAttributeNS(namespace Namespace, localname string) *Attribute {
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
func (e TemplateElement) Attributes() []*Attribute {
	return e.attributes
}
func (e *TemplateElement) Remove() {
	if e.parent != nil {
		e.parent.RemoveChild(e)
	}
}
func (e *TemplateElement) RemoveChild(node Node) Node {
	idx := slices.Index(e.templateContents, node)
	if idx != -1 {
		return nil
	}

	removed := e.templateContents[idx]
	e.templateContents = slices.Delete(e.templateContents, idx, idx+1)

	return removed
}

func (e TemplateElement) Classes() utils.StringOption {
	attr := e.GetAttribute("class")
	if attr != nil {
		return utils.Some(attr.Value)
	}

	return utils.None[string]()
}
func (e TemplateElement) Id() utils.StringOption {
	attr := e.GetAttribute("id")
	if attr != nil {
		return utils.Some(attr.Value)
	}

	return utils.None[string]()
}
