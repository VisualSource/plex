package script

import (
	"fmt"
)

type Parser struct {
	tokens []Token
	pos    int
}

func NewParser(tokens []Token) Parser {
	return Parser{
		tokens: tokens,
	}
}

func (p *Parser) Parse() (AstNode, error) {
	return p.parseTerm()
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return NewDataToken(TokenType_EOF, Position{}, Position{})
	}
	return p.tokens[p.pos]
}

func (p *Parser) peekN(n int) Token {
	if p.pos+n >= len(p.tokens) {
		return NewDataToken(TokenType_EOF, Position{}, Position{})
	}

	return p.tokens[p.pos+n]
}

func (p *Parser) advance() Token {
	t := p.peek()
	p.pos++
	return t
}

func (p *Parser) expect(tt TokenType) (Token, error) {
	if p.peek().IsToken() != tt {
		return nil, fmt.Errorf("expected %v, get %v", tt, p.peek().IsToken())
	}
	return p.advance(), nil
}

// term ::= NUMBER | IDENT | "(" expression ")"
func (p *Parser) parsePrimary() (AstNode, error) {
	t := p.peek()
	switch t.IsToken() {
	case TokenType_Number:
		tok := p.advance().(*ValueToken)
		return NewNumberLiteral(tok.Value, tok.Start, tok.End), nil
	case TokenType_Ident:
		tok := p.advance().(*ValueToken)
		return NewIdentifier(tok.Value, tok.Start, tok.End), nil
	case TokenType_BracketParamOpen:
		p.advance()
		inner, err := p.parseTerm()
		if err != nil {
			return nil, err
		}

		_, err = p.expect(TokenType_BracketParamClose)
		return inner, err
	default:
		return nil, fmt.Errorf("unexpected token: %v", t.IsToken())
	}
}

// expression := term ( "+" term )*
func (p *Parser) parseTerm() (AstNode, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for p.peek().IsToken() == TokenType_Plus {
		op := p.advance()
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}

		start, _ := left.Range()
		_, end := right.Range()

		left = &BinaryExpression{
			Start:    start,
			End:      end,
			Left:     left,
			Right:    right,
			Operator: op.IsToken(),
		}
	}

	return left, nil
}

// factor ::= unary ( ("*" | "/" | "%") unary )*
func (p *Parser) parseFactor() (AstNode, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		tt := p.peek().IsToken()

		if tt != TokenType_Star && tt != TokenType_Div && tt != TokenType_Mod {
			break
		}

		op := p.advance()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}

		start, _ := left.Range()
		_, end := right.Range()

		left = &BinaryExpression{
			Start:    start,
			End:      end,
			Operator: op.IsToken(),
		}
	}

	return left, nil
}

// unary ::= ("-"|"+") unary | primary
func (p *Parser) parseUnary() (AstNode, error) {
	tt := p.peek().IsToken()

	if tt == TokenType_Minus || tt == TokenType_Plus {
		op := p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}

		start, _ := op.Range()
		_, end := operand.Range()

		return &UnaryExpression{
			Start:    start,
			End:      end,
			Operator: op.IsToken(),
			Operand:  operand,
		}, nil
	}

	return p.parsePrimary()
}

// assignment ::= IDENT "=" expression | ternary
func (p *Parser) parseAssignment() (AstNode, error) {
	if p.peek().IsToken() == TokenType_Ident && p.peekN(1).IsToken() == TokenType_Equal {
		name := p.advance().(*ValueToken)
		p.advance() // eat "="

		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		_, end := value.Range()
		return &AssignmentExpression{
			Start: name.Start,
			End:   end,
			Name:  name.Value,
			Value: value,
		}, nil
	}

	return p.parseTernary()
}

func (p *Parser) parseExpression() (AstNode, error) {
	return nil, nil
}

// ternary ::= expression "?" expression | ternary ":" expression | ternary
func (p *Parser) parseTernary() (AstNode, error) {
	return nil, nil
}
