package utils

// PointerOf get pointer of value.
func PointerOf[T any](value T) *T {
	return &value
}

// SafeValue get value of pointer or return empty value if pointer is nil.
func SafeValue[T any](value *T) T {
	if value == nil {
		var v T
		return v
	} else {
		return *value
	}
}

// ValueOf get value of pointer or return fallback if value is nil.
func ValueOf[T any](value *T, fallback T) T {
	if value == nil {
		return fallback
	} else {
		return *value
	}
}

// Alter returns value of pointer or return fallback if value is nil or zero.
func Alter[T comparable](value *T, fallback T) T {
	var zero T
	if value == nil || *value == zero {
		return fallback
	} else {
		return *value
	}
}

// NullableOf return nil if value is zero.
func NullableOf[T comparable](v *T) *T {
	var zero T
	if v == nil || *v == zero {
		return nil
	}
	return v
}

// IsEmpty check if pointer is nil or zero.
func IsEmpty[T comparable](value *T) bool {
	var zero T
	return value == nil || *value == zero
}

// IsSame check if two pointer value are equal.
func IsSame[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	} else if a == nil || b == nil {
		return false
	} else {
		return *a == *b
	}
}
