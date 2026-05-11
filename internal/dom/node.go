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
func (d *DocumentType) SetParent(parent Node) {
	d.parent = parent
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
func (t *Text) SetParent(node Node) {
	t.parent = node
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
	return &Attribute{
		NamespaceUri: namespace,
		Prefix:       utils.None[string](),
		LocalName:    name,
		Value:        value,
	}
}
