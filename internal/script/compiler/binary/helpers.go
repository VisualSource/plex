package binary_wasm

import (
	"fmt"

	"github.com/VisualSource/plex/internal/script"
)

func collectSignatures(p *script.Program) ([]funcSig, []*script.FunctionDeclaration, []exportEntry, error) {
	var sigs []funcSig
	var funcs []*script.FunctionDeclaration
	var exports []exportEntry

	for _, stmt := range p.Stmts {
		switch n := stmt.(type) {
		case *script.FunctionDeclaration:
			sig, err := signatureOf(n, "")
			if err != nil {
				return nil, nil, nil, err
			}
			exports = append(exports, exportEntry{
				name: n.Name,
				kind: ExportFunc,
				idx:  uint32(len(funcs)), // funcidx == position in funcs
			})

			sigs = append(sigs, sig)
			funcs = append(funcs, n)
		case *script.StructImplStatement:
			for _, m := range n.Methods {
				sig, err := signatureOf(m, n.Name)
				if err != nil {
					return nil, nil, nil, err
				}
				sigs = append(sigs, sig)
				funcs = append(funcs, m)
			}
		}
	}

	return sigs, funcs, exports, nil
}

func signatureOf(f *script.FunctionDeclaration, implName string) (funcSig, error) {
	var params []byte
	if implName != "" {
		params = append(params, ValI32) // struct method implicit self
	}
	for _, p := range f.Params {
		t := p.GetType()
		if t == nil {
			return funcSig{}, fmt.Errorf("no type on param %s", p.Name)
		}
		params = append(params, valType(t))
	}

	var results []byte
	if f.ReturnType != nil {
		results = append(results, valType(f.ReturnType.GetType()))
	}

	name := f.Name
	if implName != "" {
		name = fmt.Sprintf("__%s__%s", implName, f.Name)
	}

	isMethod := implName != ""

	return funcSig{name, params, results, isMethod}, nil
}

func collectStrings(p *script.Program) *stringTable {
	st := new(stringTable)
	var walk func(script.AstNode)
	walk = func(n script.AstNode) {
		switch v := n.(type) {
		case *script.StringLiteral:
			st.add(v.Value)
		case *script.Program:
			for _, s := range v.Stmts {
				walk(s)
			}
		case *script.FunctionDeclaration:
			walk(v.Body)
		case *script.Block:
			for _, sg := range v.Stmts {
				walk(sg)
			}
		case *script.ReturnStatement:
			if v.Value != nil {
				walk(v.Value)
			}
		case *script.VariableDeclaration:
			if v.Init != nil {
				walk(v.Init)
			}
		case *script.BinaryExpression:
			walk(v.Left)
			walk(v.Right)
		case *script.IfStatement:
			walk(v.Condition)
			walk(v.Body)
			if v.Else != nil {
				walk(v.Else)
			}
		case *script.FunctionCall:
			walk(v.Callee)
			for _, a := range v.Args {
				walk(a)
			}
		case *script.MemberAccess:
			walk(v.Object)
		}
	}

	walk(p)
	return st
}
