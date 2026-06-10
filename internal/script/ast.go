package script

import (
	"fmt"
)

type Parser struct {
	tokens []Token
	pos    int
}

// #region utils
func NewParser(tokens []Token) Parser {
	return Parser{
		tokens: tokens,
	}
}

func (p *Parser) Parse() (AstNode, error) {
	return p.parseExpression()
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

func (p *Parser) binaryExpr(builder func() (AstNode, error), check func(TokenType) bool) (AstNode, error) {
	left, err := builder()
	if err != nil {
		return nil, err
	}

	for {
		tt := p.peek().IsToken()

		if !check(tt) {
			break
		}

		op := p.advance()
		right, err := builder()
		if err != nil {
			return nil, err
		}

		start, _ := left.Range()
		_, end := right.Range()

		left = &BinaryExpression{
			Start:    start,
			End:      end,
			Operator: op.IsToken(),
			Right:    right,
			Left:     left,
		}
	}

	return left, nil
}

//#endregion

// primary ::= NUMBER | STRING | IDENT | "(" expression ")"
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
		inner, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		_, err = p.expect(TokenType_BracketParamClose)
		return inner, err
	default:
		return nil, fmt.Errorf("unexpected token: %v", t.IsToken())
	}
}

// unary   ::= ( "-" | "+" ) unary | primary
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

// factor ::= unary ( ( "*" | "/" | "%" ) unary )*
func (p *Parser) parseFactor() (AstNode, error) {
	return p.binaryExpr(p.parseUnary, func(tt TokenType) bool {
		return tt == TokenType_Star || tt == TokenType_Div || tt == TokenType_Mod
	})
}

// term ::= factor ( ( "+" | "-" ) factor )*
func (p *Parser) parseTerm() (AstNode, error) {
	return p.binaryExpr(p.parseFactor, func(tt TokenType) bool {
		return tt == TokenType_Plus || tt == TokenType_Minus
	})
}

// comparison ::= term ( ( "<" | ">" | "<=" | ">=" ) term )*
func (p *Parser) parseComparison() (AstNode, error) {
	return p.binaryExpr(p.parseTerm, func(tt TokenType) bool {
		return tt == TokenType_LessThen || tt == TokenType_LessThenOrEqual || tt == TokenType_GreaterThen || tt == TokenType_GreaterThenOrEqaul
	})
}

// equality ::= comparison ( ("==" | "!=") comparison )*
func (p *Parser) parseEquality() (AstNode, error) {
	return p.binaryExpr(p.parseComparison, func(tt TokenType) bool {
		return tt == TokenType_EqualEqual || tt == TokenType_NotEqual
	})
}

// logicAnd  ::= equality ( "&&" equality )*
func (p *Parser) parseLogicAnd() (AstNode, error) {
	return p.binaryExpr(p.parseEquality, func(tt TokenType) bool {
		return tt == TokenType_AND
	})
}

// logicOr   ::= logicAnd ( "||" logicAnd )*
func (p *Parser) parseLogicOr() (AstNode, error) {
	return p.binaryExpr(p.parseLogicAnd, func(tt TokenType) bool {
		return tt == TokenType_OR
	})
}

// ternary ::= logicOr ( "?" expression ":" expression )?
func (p *Parser) parseTernary() (AstNode, error) {
	left, err := p.parseLogicOr()
	if err != nil {
		return nil, err
	}

	if p.peek().IsToken() == TokenType_Question {
		p.advance() // eat "?"

		trueBlock, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		_, err = p.expect(TokenType_Colon)
		if err != nil {
			return nil, err
		}

		falseBlock, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		start, _ := left.Range()
		_, end := falseBlock.Range()
		return &TernaryExpression{
			Start:      start,
			End:        end,
			Condition:  left,
			FalseBlock: falseBlock,
			TrueBlock:  trueBlock,
		}, nil
	}

	return left, nil
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

// functionCall ::= IDENT "(" ( <expression> ( "," <expression> ) )? ")"
func (p *Parser) parseFunctionCall() (AstNode, error) {
	if p.peek().IsToken() == TokenType_Ident && p.peekN(1).IsToken() == TokenType_BracketParamOpen {
		name := p.advance().(*ValueToken)
		p.advance() // eat "("

		args := make([]AstNode, 0)

		for {
			value, err := p.parseExpression()
			if err != nil {
				return nil, err
			}

			args = append(args, value)

			if p.peek().IsToken() == TokenType_Comma {
				p.advance()
				continue
			} else if p.peek().IsToken() == TokenType_BracketParamClose {
				break
			}

			return nil, fmt.Errorf("unexpected token %v", p.peek().IsToken())
		}

		e, err := p.expect(TokenType_BracketParamOpen)
		if err != nil {
			return nil, err
		}

		_, end := e.Range()

		return &FunctionCall{
			Start: name.Start,
			End:   end,
			Name:  name.Value,
			Args:  args,
		}, nil
	}

	return nil, nil
}

// expression ::= assignment
func (p *Parser) parseExpression() (AstNode, error) {
	return p.parseAssignment()
}
