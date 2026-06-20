package typeschecker

import (
	"cmp"
	"maps"
	"slices"

	"github.com/VisualSource/plex/internal/script"
)

type orderedItem struct {
	Type *script.Type
	Pos  int
}

type StructInfo struct {
	Fields  map[string]*orderedItem
	Methods map[string]*FuncInfo
}

func (s *StructInfo) GetFieldsInOrder() []*orderedItem {
	items := slices.Collect(maps.Values(s.Fields))

	slices.SortFunc(items, func(a, b *orderedItem) int {
		return cmp.Compare(a.Pos, b.Pos)
	})

	return items
}

func (s *StructInfo) GetTypeOf(ident string) *script.Type {
	if t, ok := s.Fields[ident]; ok {
		return t.Type
	}

	return nil
}

type FuncInfo struct {
	Scope      *Scope
	ReturnType *script.Type
	Args       map[string]*orderedItem
}

func (f *FuncInfo) GetArgsInOrder() []*orderedItem {
	items := slices.Collect(maps.Values(f.Args))

	slices.SortFunc(items, func(a, b *orderedItem) int {
		return cmp.Compare(a.Pos, b.Pos)
	})

	return items
}

func newFuncInfo(parent *Scope) *FuncInfo {
	return &FuncInfo{
		Scope: newScope(parent),
		Args:  make(map[string]*orderedItem),
	}
}

type Scope struct {
	parent *Scope
	vars   map[string]*script.Type
}

func newScope(parent *Scope) *Scope {
	return &Scope{
		parent: parent,
		vars:   make(map[string]*script.Type),
	}
}

func (s *Scope) Set(ident string, ty *script.Type) {
	s.vars[ident] = ty
}

func (s *Scope) Get(ident string) *script.Type {
	if t, ok := s.vars[ident]; ok {
		return t
	}

	if s.parent == nil {
		return nil
	}

	return s.parent.Get(ident)
}
