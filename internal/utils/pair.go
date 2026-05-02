package utils

type Pair[A any, B any] struct {
	First  A
	Second B
}

func NewPair[A any, B any](k A, v B) Pair[A, B] {
	return Pair[A, B]{First: k, Second: v}
}
