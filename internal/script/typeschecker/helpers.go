package typeschecker

import "github.com/VisualSource/plex/internal/script"

// normalizeKind collapses alias pairs that map to the same WASM primitive.
func normalizeKind(k script.TypeKind) script.TypeKind {
	switch k {
	case script.TypeKind_Float, script.TypeKind_F64:
		return script.TypeKind_F64
	case script.TypeKind_Int, script.TypeKind_I64:
		return script.TypeKind_I64
	default:
		return k
	}
}

func isSameType(a, b *script.Type) bool {
	if a == nil || b == nil {
		return false
	}

	if normalizeKind(a.Kind) != normalizeKind(b.Kind) || a.Struct != b.Struct {
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

func isNumeric(t *script.Type) bool {
	switch t.Kind {
	case script.TypeKind_I32, script.TypeKind_I64,
		script.TypeKind_F32, script.TypeKind_F64,
		script.TypeKind_Int, script.TypeKind_Float:
		return true
	}
	return false
}
