package layouts

import (
	"github.com/VisualSource/plex/internal/css/cssom"
	"github.com/VisualSource/plex/internal/utils"
)

func getDisplayValue(v utils.Option[cssom.Display]) (OuterBoxType, InnerBoxType) {
	if v.IsNone() {
		return OuterBoxType_Block, InnerBoxType_Flow
	}

	d := v.Value

	outer := OuterBoxType_Block
	inner := InnerBoxType_Flow

	switch d.Outer {
	case "inline":
		outer = OuterBoxType_Inline
	}

	switch d.Inner {
	case "flow-root":
		inner = InnerBoxType_FlowRoot
	}

	return outer, inner
}
