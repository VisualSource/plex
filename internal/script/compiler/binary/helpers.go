package binary_wasm

import (
	"fmt"

	"github.com/VisualSource/plex/internal/script"
)

func collectSignatures(p *script.Program) ([]funcSig, []*script.FunctionDeclaration, error) {
	var out []funcSig
	var funcs []*script.FunctionDeclaration
	for _, stmt := range p.Stmts {
		switch n := stmt.(type) {
		case *script.FunctionDeclaration:
			sig, err := signatureOf(n, "")
			if err != nil {
				return nil, nil, err
			}
			out = append(out, sig)

			funcs = append(funcs, n)
		case *script.StructImplStatement:
			for _, m := range n.Methods {
				sig, err := signatureOf(m, m.Name)
				if err != nil {
					return nil, nil, err
				}
				out = append(out, sig)
				funcs = append(funcs, m)
			}
		}
	}

	return out, funcs, nil
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

	return funcSig{name, params, results}, nil
}
