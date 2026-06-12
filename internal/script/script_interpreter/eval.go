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
				if ret, ok := err.(returnSignal); ok {
					return ret.value, nil
				}

				return nil, err
			}

		}

		return NullValue{}, nil
	case *script.FunctionCall:
		calleeVal, err := Eval(n.Callee, env)
		if err != nil {
			return nil, err
		}
		fn, ok := calleeVal.(FunctionValue)
		if !ok {
			return nil, fmt.Errorf("not a function")
		}

		callEnv := NewEnvironment(env)
		for i, param := range fn.Params {
			p := param.(*script.Parameter)
			arg, err := Eval(n.Args[i], env)
			if err != nil {
				return nil, err
			}
			callEnv.Set(p.Name, arg)
		}

		_, err = Eval(fn.Body, callEnv)
		if err != nil {
			if ret, ok := err.(returnSignal); ok {
				return ret.value, nil
			}
			return nil, err
		}

		return NullValue{}, nil
	default:
		return nil, fmt.Errorf("eval: unhandled node %l", node)
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
