package scriptinterpreter

import (
	"fmt"
	"strings"
)

type Environment struct {
	vars   map[string]Value
	parent *Environment
}

func NewEnvironment(parent *Environment) *Environment {
	return &Environment{
		vars: make(map[string]Value), parent: parent,
	}
}

func (e *Environment) Get(name string) (Value, bool) {
	v, ok := e.vars[name]
	if !ok && e.parent != nil {
		return e.parent.Get(name)
	}
	return v, ok
}

func (e *Environment) Set(name string, v Value) {
	e.vars[name] = v
}

func (e *Environment) Assign(name string, v Value) bool {
	if _, ok := e.vars[name]; ok {
		e.vars[name] = v
		return true
	}

	if e.parent != nil {
		return e.parent.Assign(name, v)
	}

	return false
}

func NewGlobalEnv() *Environment {
	env := NewEnvironment(nil)

	env.Set("print", NativeFunction{
		Name: "print",
		Fn: func(args []Value) (Value, error) {
			parts := make([]string, len(args))
			for i, a := range args {
				parts[i] = a.String()
			}

			fmt.Println(strings.Join(parts, " "))
			return NullValue{}, nil
		},
	})

	env.Set("str", NativeFunction{Name: "str", Fn: func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("str: expected 1 arg")
		}
		return StringValue{V: args[0].String()}, nil
	}})

	env.Set("int", NativeFunction{Name: "int", Fn: func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("str: expected 1 arg")
		}

		if s, ok := args[0].(StringValue); ok {
			if len(s.V) == 1 {
				return NumberValue{V: float64(s.V[0])}, nil
			}
		}

		return nil, fmt.Errorf("int: unable to convert to int")
	}})

	env.Set("nil", NullValue{})
	env.Set("false", BoolValue{V: false})
	env.Set("true", BoolValue{V: true})

	return env

}
