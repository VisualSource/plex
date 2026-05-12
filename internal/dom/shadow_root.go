package dom

import (
	"slices"
)

type ShadowRoot struct {
	children                      []Node
	host                          *Element
	Mode                          string // "open" | "closed"
	Clonable                      bool
	Serializable                  bool
	DelegatesFocus                bool
	SlotAssignment                string // "named" | "manual"
	Declarative                   bool
	AvailableToElementInternals   bool
	KeepCustomElementRegistryNull bool
	document                      *Document
	parent                        Node
}

func NewShadowRoot(
	document *Document,
	host *Element,
	mode string,
	slotAssignment string,
	clonable bool,
	serializable bool,
	delegatesFocus bool,
	keepRegistryNull bool,
) *ShadowRoot {
	return &ShadowRoot{
		children:                      make([]Node, 0),
		host:                          host,
		Mode:                          mode,
		SlotAssignment:                slotAssignment,
		Clonable:                      clonable,
		Serializable:                  serializable,
		DelegatesFocus:                delegatesFocus,
		KeepCustomElementRegistryNull: keepRegistryNull,
		document:                      document,
		parent:                        host,
	}
}

func (s *ShadowRoot) Host() *Element { return s.host }

func (s ShadowRoot) IsNode() uint         { return 11 }
func (s ShadowRoot) Tag() string          { return "#shadow-root" }
func (s ShadowRoot) Namespace() Namespace { return NamespaceHTML }
func (s ShadowRoot) Parent() Node         { return s.parent }
func (s *ShadowRoot) SetParent(node Node) {
	s.parent = node
}
func (s ShadowRoot) Document() *Document { return s.document }
func (s ShadowRoot) Children() []Node    { return s.children }

func (s ShadowRoot) PreviousSibling() Node { return nil }
func (s ShadowRoot) Remove()               {}
func (s *ShadowRoot) RemoveChild(node Node) Node {
	idx := slices.Index(s.children, node)
	if idx == -1 {
		return nil
	}
	removed := s.children[idx]
	s.children = slices.Delete(s.children, idx, idx+1)
	return removed
}

func (s *ShadowRoot) AppendChild(node Node) {
	adoptNode(node, s)
	node.SetParent(s)
	s.children = append(s.children, node)
}

func (s *ShadowRoot) PrependChild(node Node) {
	adoptNode(node, s)
	node.SetParent(s)
	s.children = slices.Insert(s.children, 0, node)
}

func (s *ShadowRoot) InsertBefore(node Node, ref Node) {
	adoptNode(node, s)
	node.SetParent(s)
	idx := slices.Index(s.children, ref)
	if idx == -1 {
		s.children = append(s.children, node)
	} else {
		s.children = slices.Insert(s.children, idx, node)
	}
}
