package scriptinterpreter

import (
	"fmt"
	"slices"

	"github.com/VisualSource/plex/internal/script"
)

type Value interface {
	String() string
	Truthy() bool
}

type NumberValue struct{ V float64 }
type StringValue struct{ V string }
type BoolValue struct{ V bool }
type NullValue struct{}

func (n NumberValue) Truthy() bool { return n.V != 0 }
func (s StringValue) Truthy() bool { return s.V != "" }
func (b BoolValue) Truthy() bool   { return b.V }
func (n NullValue) Truthy() bool   { return false }

func (n NumberValue) String() string { return fmt.Sprintf("%g", n.V) }
func (s StringValue) String() string { return s.V }
func (b BoolValue) String() string   { return fmt.Sprintf("%v", b.V) }
func (n NullValue) String() string   { return "null" }

type FunctionValue struct {
	Params []*script.Parameter
	Body   script.AstNode
	Env    *Environment
}

func (f FunctionValue) Truthy() bool   { return true }
func (f FunctionValue) String() string { return "<fn>" }

type StructValue struct {
	TypeName string
	Fields   map[string]Value
}

func (s *StructValue) Truthy() bool   { return true }
func (s *StructValue) String() string { return fmt.Sprintf("%s {...}", s.TypeName) }

type StructType struct {
	Name    string
	Fields  []string
	Methods map[string]FunctionValue
}

func (s StructType) Truthy() bool   { return true }
func (s StructType) String() string { return fmt.Sprintf("<type:%s>", s.Name) }

type NativeFunction struct {
	Name string
	Fn   func(args []Value) (Value, error)
}

func (n NativeFunction) Truthy() bool   { return true }
func (n NativeFunction) String() string { return fmt.Sprintf("<native:%s>", n.Name) }

type ArrayValue struct {
	Elements []Value
}

func (a *ArrayValue) Truthy() bool   { return len(a.Elements) > 0 }
func (a *ArrayValue) String() string { return fmt.Sprintf("<array len=%d>", len(a.Elements)) }

func (a *ArrayValue) len() (NumberValue, error) {
	return NumberValue{V: float64(len(a.Elements))}, nil
}

func (a *ArrayValue) append(value Value) (NullValue, error) {
	a.Elements = append(a.Elements, value)

	return NullValue{}, nil
}

func (a *ArrayValue) remove(arg Value) (Value, error) {
	idx, ok := arg.(NumberValue)
	if !ok {
		return nil, fmt.Errorf("expected argument to be a number")
	}
	num := int(idx.V)

	removed := a.Elements[num]

	a.Elements = slices.Delete(a.Elements, num, num+1)

	return removed, nil
}
