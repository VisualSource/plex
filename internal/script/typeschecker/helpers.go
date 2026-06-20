package typeschecker

import "github.com/VisualSource/plex/internal/script"

func isSameType(a, b *script.Type) bool {
	if a == nil || b == nil {
		return false
	}

	if a.Kind != b.Kind || a.Struct != b.Struct {
		return false
	}

	if a.Element != nil || b.Element != nil {
		if a.Element == nil || b.Element == nil {
			return false
		}

		return isSameType(a.Element, b.Element)
	}

	return true
}
