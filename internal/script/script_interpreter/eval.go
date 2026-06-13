package scriptinterpreter

import (
	"fmt"
	"strconv"

	"github.com/VisualSource/plex/internal/script"
)

func Eval(node script.AstNode, env *Environment) (Value, error) {
	switch n := node.(type) {
	case *script.Program:
		var result Value = NullValue{}
		for _, stmts := range n.Stmts {
			v, err := Eval(stmts, env)
			if err != nil {
				return nil, err
			}
			result = v
		}
		return result, nil
	case *script.StringLiteral:
		return StringValue{V: n.Value}, nil
	case *script.NumberLiteral:
		f, err := strconv.ParseFloat(n.Value, 64)
		if err != nil {
			return nil, err
		}

		return NumberValue{V: f}, nil
	case *script.Identifier:
		v, ok := env.Get(n.Value)
		if !ok {
			return nil, fmt.Errorf("undefined: %s", n.Value)
		}
		return v, nil
	case *script.VariableDeclaration:
		val, err := Eval(n.Init, env)
		if err != nil {
			return nil, err
		}
		env.Set(n.Name, val)
		return NullValue{}, nil
	case *script.AssignmentExpression:
		val, err := Eval(n.Value, env)
		if err != nil {
			return nil, err
		}
		if !env.Assign(n.Name, val) {
			return nil, fmt.Errorf("assign to undefined: %s", n.Name)
		}
		return val, nil
	case *script.BinaryExpression:
		return evalBinary(n, env)
	case *script.IfStatement:
		cond, err := Eval(n.Condition, env)
		if err != nil {
			return nil, err
		}

		if cond.Truthy() {
			return Eval(n.Body, env)
		} else if n.Else != nil {
			return Eval(n.Else, env)
		}
		return NullValue{}, nil
	case *script.Block:
		child := NewEnvironment(env)

		var result Value = NullValue{}
		for _, stmt := range n.Stmts {
			v, err := Eval(stmt, child)
			if err != nil {
				return nil, err
			}
			result = v
		}
		return result, nil
	case *script.BreakStatement:
		return nil, breakSignal{}
	case *script.ReturnStatement:
		if n.Value == nil {
			return nil, returnSignal{value: NullValue{}}
		}
		val, err := Eval(n.Value, env)
		if err != nil {
			return nil, err
		}
		return nil, returnSignal{value: val}

	case *script.FunctionDeclaration:
		env.Set(n.Name, FunctionValue{
			Params: n.Params,
			Body:   n.Body,
			Env:    NewEnvironment(env),
		})
		return NullValue{}, nil
	case *script.WhileStatement:

		for {
			cond, err := Eval(n.Condition, env)
			if err != nil {
				return nil, err
			}

			if !cond.Truthy() {
				break
			}

			_, err = Eval(n.Body, env)
			if err != nil {
				if _, ok := err.(breakSignal); ok {
					return NullValue{}, nil
				}

				return nil, err
			}
		}

		return NullValue{}, nil
	case *script.FunctionCall:
		if ma, ok := n.Callee.(*script.MemberAccess); ok {
			obj, err := Eval(ma.Object, env)
			if err != nil {
				return nil, err
			}

			switch instance := obj.(type) {
			case *ArrayValue:
				switch ma.Field {
				case "len":
					return instance.len()
				case "append":
					if len(n.Args) != 1 {
						return nil, fmt.Errorf("requires a arg")
					}
					arg, err := Eval(n.Args[0], env)
					if err != nil {
						return nil, err
					}

					return instance.append(arg)
				case "remove":
					if len(n.Args) != 1 {
						return nil, fmt.Errorf("requires a arg")
					}
					arg, err := Eval(n.Args[0], env)
					if err != nil {
						return nil, err
					}

					return instance.remove(arg)
				default:
					return nil, fmt.Errorf("array has not method %s", ma.Field)
				}
			case StringValue:
				if ma.Field == "len" {
					return NumberValue{V: float64(len(instance.V))}, nil
				}
				return nil, fmt.Errorf("#string does not have method %s", ma.Field)
			case *StructValue:
				typeVal, _ := env.Get(instance.TypeName)
				st := typeVal.(StructType)
				method, ok := st.Methods[ma.Field]
				if !ok {
					return nil, fmt.Errorf("no method %s", ma.Field)
				}

				callEnv := NewEnvironment(method.Env)

				for i, param := range method.Params {
					if i > len(n.Args)-1 {
						return nil, fmt.Errorf("method %s was expecting arg at position %d", ma.Field, i+1)
					}
					arg, err := Eval(n.Args[i], env)
					if err != nil {
						return nil, err
					}

					callEnv.Set(param.(*script.Parameter).Name, arg)
				}
				callEnv.Set("self", instance) // override arg name self

				_, err := Eval(method.Body, callEnv)
				if ret, ok := err.(returnSignal); ok {
					return ret.value, nil
				}
				return NullValue{}, err

			}
		}

		calleeVal, err := Eval(n.Callee, env)
		if err != nil {
			return nil, err
		}

		switch callee := calleeVal.(type) {
		case StructType:
			instance := &StructValue{
				TypeName: callee.Name,
				Fields:   make(map[string]Value),
			}

			for i, fieldName := range callee.Fields {
				arg, err := Eval(n.Args[i], env)
				if err != nil {
					return nil, err
				}
				instance.Fields[fieldName] = arg
			}
			return instance, nil
		case FunctionValue:
			callEnv := NewEnvironment(callee.Env)
			for i, param := range callee.Params {
				p := param.(*script.Parameter)
				arg, err := Eval(n.Args[i], env)
				if err != nil {
					return nil, err
				}
				callEnv.Set(p.Name, arg)
			}

			_, err = Eval(callee.Body, callEnv)
			if err != nil {
				if ret, ok := err.(returnSignal); ok {
					return ret.value, nil
				}
				return nil, err
			}
			return NullValue{}, nil
		case NativeFunction:
			args := make([]Value, len(n.Args))

			for i, arg := range n.Args {
				v, err := Eval(arg, env)
				if err != nil {
					return nil, err
				}
				args[i] = v
			}

			return callee.Fn(args)
		default:
			return nil, fmt.Errorf("not callable")
		}
	case *script.StructStatement:
		fields := make([]string, 0, len(n.Fields))
		for _, f := range n.Fields {
			fields = append(fields, f.(*script.Parameter).Name)
		}

		env.Set(n.Name, StructType{
			Name:    n.Name,
			Fields:  fields,
			Methods: make(map[string]FunctionValue),
		})
		return NullValue{}, nil
	case *script.StructImplStatement:
		v, ok := env.Get(n.Name)
		if !ok {
			return nil, fmt.Errorf("impl: unknown type %s", n.Name)
		}
		st, ok := v.(StructType)
		if !ok {
			return nil, fmt.Errorf("%s is not a struct type", n.Name)
		}
		for _, m := range n.Methods {
			fn := m.(*script.FunctionDeclaration)
			st.Methods[fn.Name] = FunctionValue{Params: fn.Params, Body: fn.Body, Env: env}
		}
		env.Assign(n.Name, st)

		return NullValue{}, nil
	case *script.MemberAccess:
		obj, err := Eval(n.Object, env)
		if err != nil {
			return nil, err
		}

		instance, ok := obj.(*StructValue)
		if !ok {
			return nil, fmt.Errorf("%s is not a struct", n.Field)
		}
		v, ok := instance.Fields[n.Field]
		if !ok {

			return nil, fmt.Errorf("no field %s", n.Field)
		}
		return v, nil
	case *script.MemberAssignment:
		obj, err := Eval(n.Object, env)
		if err != nil {
			return nil, err
		}
		instance, ok := obj.(*StructValue)
		if !ok {
			return nil, fmt.Errorf("cannot assig field on non-struct value")
		}

		val, err := Eval(n.Value, env)
		if err != nil {
			return nil, err
		}

		instance.Fields[n.Field] = val
		return val, nil
	case *script.TernaryExpression:
		cond, err := Eval(n.Condition, env)
		if err != nil {
			return nil, err
		}

		if cond.Truthy() {
			result, err := Eval(n.TrueBlock, env)
			if err != nil {
				return nil, err
			}
			return result, nil
		}

		result, err := Eval(n.FalseBlock, env)
		if err != nil {
			return nil, err
		}
		return result, nil
	case *script.ArrayAccess:
		field, err := Eval(n.Field, env)
		if err != nil {
			return nil, err
		}

		obj, err := Eval(n.Object, env)
		if err != nil {
			return nil, err
		}

		switch n := field.(type) {
		case *ArrayValue:
			i, ok := obj.(NumberValue)
			if !ok {
				return nil, fmt.Errorf("index into non-array")
			}

			idx := int(i.V)

			if idx < 0 || idx > len(n.Elements) {
				return nil, fmt.Errorf("index out of bounds")
			}

			return n.Elements[idx], nil
		case StringValue:
			num, ok := obj.(NumberValue)
			if !ok {
				return nil, fmt.Errorf("cannot index string with %T", num)
			}

			r := int(num.V)

			if r < 0 || r > len(n.V) {
				return nil, fmt.Errorf("index out of range")
			}

			return StringValue{V: string(n.V[r])}, nil
		default:
			return nil, fmt.Errorf("cannot index %T", obj)
		}
	case *script.ArrayLiteral:
		elements := make([]Value, len(n.Elements))
		for i, el := range n.Elements {
			v, err := Eval(el, env)
			if err != nil {
				return nil, err
			}
			elements[i] = v
		}

		return &ArrayValue{Elements: elements}, nil
	default:
		return nil, fmt.Errorf("eval: unhandled node %T", node)
	}
}

func evalBinary(n *script.BinaryExpression, env *Environment) (Value, error) {
	left, err := Eval(n.Left, env)
	if err != nil {
		return nil, err
	}

	right, err := Eval(n.Right, env)
	if err != nil {
		return nil, err
	}

	lv, lok := left.(NumberValue)
	rv, rok := right.(NumberValue)

	switch n.Operator {
	case script.TokenType_Plus:
		if lok && rok {
			return NumberValue{V: lv.V + rv.V}, nil
		}
		return StringValue{V: left.String() + right.String()}, nil
	case script.TokenType_Minus:
		if !lok || !rok {
			return nil, fmt.Errorf("- requires numbers")
		}
		return NumberValue{V: lv.V - rv.V}, nil
	case script.TokenType_Star:
		if !lok || !rok {
			return nil, fmt.Errorf("* requires numbers")
		}
		return NumberValue{V: lv.V * rv.V}, nil
	case script.TokenType_EqualEqual:
		return BoolValue{V: left.String() == right.String()}, nil
	case script.TokenType_LessThen:
		if !lok || !rok {
			return nil, fmt.Errorf("< requires numbers")
		}
		return BoolValue{V: lv.V < rv.V}, nil
	default:
		return nil, fmt.Errorf("unknown operator: %v", n.Operator)
	}
}

type returnSignal struct{ value Value }

func (r returnSignal) Error() string { return "return" }

type breakSignal struct{}

func (b breakSignal) Error() string { return "break" }
