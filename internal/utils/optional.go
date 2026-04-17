package utils

type Option[T comparable] struct {
	Value *T
}

type StringOption = Option[string]
type BoolOption = Option[bool]

func Some[T comparable](value T) Option[T] {
	return Option[T]{Value: new(value)}
}

func None[T comparable]() Option[T] {
	return Option[T]{Value: nil}
}

func (o Option[T]) IsNone() bool {
	return o.Value == nil
}

func (o Option[T]) IsSome() bool {
	return o.Value != nil
}

func (o Option[T]) Is(value T) bool {
	return IsSame(o.Value, &value)
}

func (o *Option[T]) Set(value T) {
	o.Value = new(value)
}

func (o *Option[T]) Alter(mod func(value *T) *T) {
	o.Value = mod(o.Value)
}

func (o Option[T]) SomeOr(value T) T {
	return ValueOf(o.Value, value)
}
