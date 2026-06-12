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
				return nil, err
			}

		}

		return NullValue{}, nil
	case *script.FunctionCall:
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
			if ma, ok := n.Callee.(*script.MemberAccess); ok {
				obj, err := Eval(ma.Object, env)
				if err != nil {
					return nil, err
				}

				instance, ok := obj.(*StructValue)
				if ok {
					typeVal, _ := env.Get(instance.TypeName)
					st := typeVal.(StructType)
					method, ok := st.Methods[ma.Field]
					if !ok {
						return nil, fmt.Errorf("no method %s", ma.Field)
					}

					callEnv := NewEnvironment(method.Env)

					for i, param := range method.Params {
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
