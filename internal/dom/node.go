package dom

import (
	"slices"
	"strings"

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

// setParent updates a node's parent pointer. Used by AppendChild/PrependChild/
// InsertBefore to keep parent backrefs consistent when nodes are moved.
func setParent(node Node, parent Node) {
	switch n := node.(type) {
	case *Element:
		n.parent = parent
	case *TemplateElement:
		n.parent = parent
	case *Text:
		n.parent = parent
	case *Comment:
		n.parent = parent
	case *DocumentType:
		n.parent = parent
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

type DocumentType struct {
	Name     string
	PublicId string
	SystemId string
	document *Document
	parent   Node
}

func (d DocumentType) Parent() Node {
	return d.parent
}
func (d DocumentType) IsNode() uint {
	return 1
}
func (d DocumentType) Tag() string {
	return "#doctype"
}
func (d DocumentType) Namespace() Namespace {
	return NamespaceHTML
}
func (d *DocumentType) AppendChild(node Node) {
	panic("DocumentType node does not support this method")
}
func (d *DocumentType) PrependChild(node Node) {
	panic("DocumentType node does not support this method")
}
func (d *DocumentType) InsertBefore(node Node, ref Node) {
	panic("DocumentType node does not support this method")
}
func (d DocumentType) Document() *Document {
	return d.document
}
func (d DocumentType) PreviousSibling() Node {
	return nil
}
func (d DocumentType) Children() []Node {
	return nil
}
func (d DocumentType) Remove() {}
func (d DocumentType) RemoveChild(Node) Node {
	return nil
}

func NewDocumentType(name string, publicId string, systemId string) *DocumentType {
	return &DocumentType{Name: name, PublicId: publicId, SystemId: systemId}
}

type Text struct {
	Data     string
	document *Document
	parent   Node
}

func (t Text) Parent() Node {
	return t.parent
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

func NewTextNode(document *Document, data string, parent Node) *Text {
	return &Text{Data: data, document: document, parent: parent}
}

type Comment struct {
	Data     string
	document *Document
	parent   Node
}

func (c Comment) Parent() Node {
	return c.parent
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
func NewComment(document *Document, parent Node, data string) *Comment {
	return &Comment{Data: data, document: document, parent: parent}
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
	prefix := utils.None[string]()

	if strings.Contains(name, ":") {
		items := strings.Split(name, ":")
		prefix.Set(items[0])
		name = items[1]
	}

	return &Attribute{
		NamespaceUri: namespace,
		Prefix:       prefix,
		LocalName:    name,
		Value:        value,
	}
}

type Element struct {
	children   []Node
	localName  string
	namespace  Namespace
	Is         utils.StringOption
	document   *Document
	attributes []*Attribute
	prefix     utils.StringOption
	parent     Node
}

func (e Element) Parent() Node {
	return e.parent
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
	setParent(node, e)
	e.children = append(e.children, node)
}
func (e *Element) PrependChild(node Node) {
	adoptNode(node, e)
	setParent(node, e)
	e.children = slices.Insert(e.children, 0, node)
}
func (e *Element) InsertBefore(node Node, ref Node) {
	adoptNode(node, e)
	setParent(node, e)
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
	parent := e.Parent()

	if parent == nil {
		return nil
	}

	children := parent.Children()
	if children == nil {
		return nil
	}

	idx := slices.IndexFunc(children, func(node Node) bool {
		return node == &e
	})

	if idx == -1 || idx-1 < 0 {
		return nil
	}

	return children[idx-1]
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

type TemplateElement struct {
	templateContents []Node
	document         *Document
	parent           Node
	attributes       []*Attribute
}

func (e TemplateElement) Document() *Document {
	return e.document
}

func (e TemplateElement) Parent() Node {
	return e.parent
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
	adoptNode(node, e)
	setParent(node, e)
	e.templateContents = append(e.templateContents, node)
}
func (e *TemplateElement) PrependChild(node Node) {
	adoptNode(node, e)
	setParent(node, e)
	e.templateContents = slices.Insert(e.templateContents, 0, node)
}
func (e *TemplateElement) InsertBefore(node Node, ref Node) {
	adoptNode(node, e)
	setParent(node, e)
	idx := slices.Index(e.templateContents, ref)
	if idx == -1 {
		e.templateContents = append(e.templateContents, node)
	} else {
		e.templateContents = slices.Insert(e.templateContents, idx, node)
	}
}
func (e TemplateElement) PreviousSibling() Node {
	parent := e.Parent()

	if parent == nil {
		return nil
	}

	children := parent.Children()
	if children == nil {
		return nil
	}

	idx := slices.IndexFunc(children, func(node Node) bool {
		return node == &e
	})

	if idx == -1 || idx-1 < 0 {
		return nil
	}

	return children[idx-1]
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

// https://dom.spec.whatwg.org/#concept-create-element
func NewElement(
	document *Document,
	localName string,
	namespace utils.Option[Namespace],
	prefix utils.StringOption,
	is utils.StringOption, synchronusCustomElement bool,
	registry utils.StringOption,
	parent Node) ElementNode {

	if utils.ValueOf(namespace.Value, NamespaceHTML) == NamespaceHTML && localName == "template" {
		return &TemplateElement{
			document:   document,
			parent:     parent,
			attributes: make([]*Attribute, 0),
		}
	}

	return &Element{
		document:   document,
		localName:  localName,
		namespace:  utils.ValueOf(namespace.Value, NamespaceHTML),
		prefix:     prefix,
		Is:         is,
		parent:     parent,
		attributes: make([]*Attribute, 0),
	}
}
