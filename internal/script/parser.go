package script

import "io"

func Parse(stream io.Reader) (*Program, error) {
	tokens, err := NewTokenizer(stream).Tokenize()
	if err != nil {
		return nil, err
	}

	parser := NewParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	return ast, nil
}
