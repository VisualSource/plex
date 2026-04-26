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
	Parent() Node
	Document() *Document
}

type DocumentType struct {
	Name     string
	PublicId string
	SystemId string
	document *Document
}

func (d DocumentType) Parent() Node {
	return nil
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
func (d DocumentType) Document() *Document {
	return d.document
}

func NewDocumentType(name string, publicId string, systemId string) *DocumentType {
	return &DocumentType{Name: name, PublicId: publicId, SystemId: systemId}
}

type Text struct {
	Data     string
	document *Document
}

func (t Text) Parent() Node {
	return nil
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
func (t Text) Document() *Document {
	return t.document
}

func NewTextNode(data string) *Text {
	return &Text{Data: data}
}

type Comment struct {
	Data     string
	document *Document
}

func (c Comment) Parent() Node {
	return nil
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
func (c Comment) Document() *Document {
	return c.document
}
func NewComment(data string) *Comment {
	return &Comment{Data: data}
}

type Attribute struct {
	NamespaceUri Namespace
	Prefix       utils.StringOption
	LocalName    string
	Value        string
}

func NewAttribute(namespace Namespace, prefix utils.StringOption, name string, value string) Attribute {
	return Attribute{
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
	attributes []Attribute
	prefix     utils.StringOption
}

func (e Element) Parent() Node {
	return nil
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
	e.children = append(e.children, node)
}
func (e *Element) PrependChild(node Node) {
	e.children = slices.Insert(e.children, 0, node)
}
func (e Element) Document() *Document {
	return e.document
}
func (e *Element) SetAttribute(key string, value string) {
	prefix := utils.None[string]()
	name := key
	namespace := NamespaceHTML

	if strings.Contains(key, ":") {
		items := strings.Split(key, ":")

		prefix.Set(items[0])
		name = items[1]

		switch *prefix.Value {
		case "xmlns":
			namespace = NamespaceXMLNS
		case "xlink":
			namespace = NamespaceXLink
		}
	}

	e.attributes = append(e.attributes, NewAttribute(namespace, prefix, name, value))
}
func (e Element) GetAttribute(key string) *Attribute {
	name := strings.ToLower(key)

	for _, attr := range e.attributes {
		qualName := attr.LocalName
		if attr.Prefix.IsSome() {
			qualName = *attr.Prefix.Value + ":" + attr.LocalName
		}

		if qualName == name {
			return &attr
		}
	}

	return nil
}

func (e Element) GetAttributeNS(namespace Namespace, localname string) *Attribute {
	for _, attr := range e.attributes {
		if attr.NamespaceUri == namespace && attr.LocalName == localname {
			return &attr
		}
	}

	return nil
}

func (e Element) HasAttribute(key string) bool {
	attr := e.GetAttribute(key)
	return attr != nil
}

// https://dom.spec.whatwg.org/#concept-create-element
func NewElement(document *Document, localName string, namespace utils.Option[Namespace], prefix utils.StringOption, is utils.StringOption, synchronusCustomElement bool, registry utils.StringOption) *Element {
	return &Element{
		document:  document,
		localName: localName,
		namespace: utils.ValueOf(namespace.Value, NamespaceHTML),
		prefix:    prefix,
		Is:        is,
	}
}

type TemplateElement struct {
	templateContents []Node
	document         *Document
}

func (e TemplateElement) Document() *Document {
	return e.document
}

func (e TemplateElement) Parent() Node {
	return nil
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
	e.templateContents = append(e.templateContents, node)
}
func (e *TemplateElement) PrependChild(node Node) {
	e.templateContents = slices.Insert(e.templateContents, 0, node)
}

func NewTemplateElement() *TemplateElement {
	return &TemplateElement{}
}
