package layout

import (
	"github.com/VisualSource/plex/internal/layout/styletree"
)

func NewLayoutTree(node *styletree.StyledNode) Box {

	box := Box{
		OuterBoxType: OuterBoxType_Block,
	}

	return box
}

/*func (*Layout) Paint(renderer *sdl.Renderer, window *sdl.Window, ctx context.Context) error {

	return nil
}*/
