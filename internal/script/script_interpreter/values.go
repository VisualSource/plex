package scriptinterpreter

import (
	"fmt"

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
	Params []script.AstNode
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
