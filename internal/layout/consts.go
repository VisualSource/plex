package layout

import (
	"github.com/VisualSource/plex/internal/layout/styletree"
	"github.com/Zyko0/go-sdl3/sdl"
)

type Edge struct {
	Top    float32
	Left   float32
	Right  float32
	Bottom float32
}

type Dimensions struct {
	Content sdl.FRect
	Padding Edge
	border  Edge
	margin  Edge
}

type OuterBoxType uint
type InnerBoxType uint

const (
	OuterBoxType_Block OuterBoxType = iota
	OuterBoxType_Inline
	OuterBoxType_RunIn
)

type Box struct {
	Dimensions   Dimensions
	OuterBoxType OuterBoxType
	InnerBoxType InnerBoxType
	Node         *styletree.StyledNode
	Children     []Box
}
