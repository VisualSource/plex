package script

import (
	"fmt"
)

type Parser struct {
	tokens []Token
	pos    int
}

// #region Entry

// program ::= statement* EOF
func (p *Parser) Parse() (AstNode, error) {
	program := &Program{}

	for p.peek().IsToken() != TokenType_EOF {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}

		program.Stmts = append(program.Stmts, stmt)
	}

	return program, nil
}

//#endregion

// #region utils
func NewParser(tokens []Token) Parser {
	return Parser{
		tokens: tokens,
	}
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

var tokenSymbol = map[TokenType]string{
	TokenType_Plus:               "+",
	TokenType_Minus:              "-",
	TokenType_Star:               "*",
	TokenType_Div:                "/",
	TokenType_Mod:                "%",
	TokenType_LessThen:           "<",
	TokenType_GreaterThen:        ">",
	TokenType_LessThenOrEqual:    "<=",
	TokenType_GreaterThenOrEqaul: ">=",
	TokenType_EqualEqual:         "==",
	TokenType_NotEqual:           "!=",
	TokenType_AND:                "&&",
	TokenType_OR:                 "||",
	TokenType_Incrment:           "++",
	TokenType_Decrement:          "--",
	TokenType_Equal:              "=",
	TokenType_EOF:                "EOF",
	TokenType_BracketCurlyOpen:   "{",
	TokenType_BracketCulryClose:  "}",
	TokenType_BracketSquareOpen:  "[",
	TokenType_BracketSquareClose: "]",
	TokenType_BracketParamOpen:   "(",
	TokenType_BracketParamClose:  ")",
	TokenType_Semicolon:          ";",
	TokenType_Colon:              ":",
	TokenType_Comma:              ",",
	TokenType_Dot:                ".",
	TokenType_Keyword:            "#keyword",
	TokenType_String:             "#string",
	TokenType_Ident:              "#ident",
	TokenType_Question:           "?",
	TokenType_FatArrow:           "=>",
	TokenType_Number:             "#number",
}

func (p *Parser) expect(tt TokenType) (Token, error) {
	if p.peek().IsToken() != tt {
		return nil, fmt.Errorf("expected %v, get %v", tokenSymbol[tt], tokenSymbol[p.peek().IsToken()])
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

//#region Expressions

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
	case TokenType_String:
		tok := p.advance().(*ValueToken)
		return NewStringLiteral(tok.Value, tok.Start, tok.End), nil
	case TokenType_BracketParamOpen:
		p.advance()
		inner, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		_, err = p.expect(TokenType_BracketParamClose)
		return inner, err
	case TokenType_BracketSquareOpen:
		open := p.advance()
		start, _ := open.Range()
		elements := make([]AstNode, 0)
		for p.peek().IsToken() != TokenType_BracketSquareClose {
			el, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			elements = append(elements, el)
			if p.peek().IsToken() != TokenType_Comma {
				break
			}
			p.advance() // ","
		}

		close, err := p.expect(TokenType_BracketSquareClose)
		if err != nil {
			return nil, err
		}

		_, end := close.Range()

		return &ArrayLiteral{
			Start:    start,
			End:      end,
			Elements: elements,
		}, nil
	default:
		return nil, fmt.Errorf("unexpected token: %v", tokenSymbol[t.IsToken()])
	}
}

func (p *Parser) parseArguments() ([]AstNode, error) {
	args := make([]AstNode, 0)

	if p.peek().IsToken() == TokenType_BracketParamClose {
		return args, nil
	}

	for {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		args = append(args, arg)
		if p.peek().IsToken() != TokenType_Comma {
			break
		}
		p.advance()
	}

	return args, nil
}

// postfix ::= primary (  "(" args? ")" | "." IDENT | "[" expr "]" | "++" | "--"  )
func (p *Parser) parsePostfix() (AstNode, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		switch p.peek().IsToken() {
		case TokenType_BracketParamOpen:
			p.advance()
			args, err := p.parseArguments()
			if err != nil {
				return nil, err
			}

			close, err := p.expect(TokenType_BracketParamClose)
			if err != nil {
				return nil, err
			}

			start, _ := expr.Range()
			_, end := close.Range()
			expr = &FunctionCall{
				Start:  start,
				End:    end,
				Callee: expr,
				Args:   args,
			}
		case TokenType_BracketSquareOpen: // array access "[" <expresion> "]"
			p.advance()

			obj, err := p.parseExpression()
			if err != nil {
				return nil, err
			}

			close, err := p.expect(TokenType_BracketSquareClose)
			if err != nil {
				return nil, err
			}

			start, _ := expr.Range()
			_, end := close.Range()

			expr = &ArrayAccess{
				Start:  start,
				End:    end,
				Object: obj,
				Field:  expr,
			}
		case TokenType_Dot:
			p.advance()
			field, err := p.expect(TokenType_Ident)
			if err != nil {
				return nil, err
			}
			start, _ := expr.Range()
			_, end := field.Range()

			expr = &MemberAccess{
				Start:  start,
				End:    end,
				Object: expr,
				Field:  field.(*ValueToken).Value,
			}

		case TokenType_Decrement, TokenType_Incrment:
			op := p.advance()
			start, _ := expr.Range()
			_, end := op.Range()

			expr = &UnaryExpression{
				Start:    start,
				End:      end,
				Operator: op.IsToken(),
				Operand:  expr,
				Postfix:  true,
			}
		default:
			return expr, nil
		}
	}

}

// unary   ::= ( "-" | "+" ) unary | postfix
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

	return p.parsePostfix()
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
	expr, err := p.parseTernary()
	if err != nil {
		return nil, err
	}

	if p.peek().IsToken() != TokenType_Equal {
		return expr, nil // not assignment
	}
	p.advance() // eat '='

	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	start, _ := expr.Range()
	_, end := value.Range()

	switch target := expr.(type) {
	case *Identifier:
		return &AssignmentExpression{
			Start: start,
			End:   end,
			Name:  target.Value,
			Value: value,
		}, nil
	case *MemberAccess:
		return &MemberAssignment{
			Start:  start,
			End:    end,
			Object: target.Object,
			Field:  target.Field,
			Value:  value,
		}, nil
	default:
		return nil, fmt.Errorf("invalid assignment target")

	}
}

// expression ::= assignment
func (p *Parser) parseExpression() (AstNode, error) {
	return p.parseAssignment()
}

//#endregion

//#region Stmts

func isKeyword(tt Token, keyword string) bool {
	if tt.IsToken() == TokenType_Keyword {
		value := tt.(*ValueToken)
		return value.Value == keyword
	}
	return false
}

// statement ::= importStmt | varDecl | ifStmt | whileStmt | fnDecl | structDecl | exprStmt
func (p *Parser) parseStatement() (AstNode, error) {
	t := p.peek()

	switch {
	case isKeyword(t, "import"):
		return p.parseImport()
	case isKeyword(t, "fn"):
		return p.parseFnDecl()
	case isKeyword(t, "struct"):
		return p.parseStruct()
	case isKeyword(t, "impl"):
		return p.parseStructImpl()
	case isKeyword(t, "if"):
		return p.parseIfStmt()
	case isKeyword(t, "while"):
		return p.parseWhile()
	case isKeyword(t, "let"):
		return p.parseVarDecl()
	case isKeyword(t, "return"):
		return p.parseReturnStatement()
	case isKeyword(t, "break"):
		return p.parseBreakStatement()
	default:
		return p.parseExprStmt()
	}
}

// block ::= "{" statement* "}"
func (p *Parser) parseBlock() (AstNode, error) {
	open, err := p.expect(TokenType_BracketCurlyOpen)
	if err != nil {
		return nil, err
	}

	start, _ := open.Range()
	stmts := make([]AstNode, 0)

	for p.peek().IsToken() != TokenType_BracketCulryClose && p.peek().IsToken() != TokenType_EOF {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
	}

	close, err := p.expect(TokenType_BracketCulryClose)
	if err != nil {
		return nil, err
	}
	_, end := close.Range()

	return &Block{
		Start: start,
		End:   end,
		Stmts: stmts,
	}, nil
}

// exprStmt ::= expression ";"
func (p *Parser) parseExprStmt() (AstNode, error) {
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(TokenType_Semicolon); err != nil {
		return nil, err
	}

	return expr, nil
}

// ifStmt ::= "if" expression block ( "else" ( ifStmt | block ) )?
func (p *Parser) parseIfStmt() (AstNode, error) {
	start, _ := p.advance().Range()

	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	node := &IfStatement{
		Start:     start,
		Condition: condition,
		Body:      body,
	}

	if isKeyword(p.peek(), "else") {
		p.advance()

		if isKeyword(p.peek(), "if") {
			node.Else, err = p.parseIfStmt()
		} else {
			node.Else, err = p.parseBlock()
		}

		if err != nil {
			return nil, err
		}
	}

	_, node.End = node.Body.Range()

	return node, nil
}

// params ::= param ( "," param )*
func (p *Parser) parseParams() ([]AstNode, error) {
	params := make([]AstNode, 0)

	for {
		if p.peek().IsToken() != TokenType_Ident && p.peek().IsToken() != TokenType_Keyword {
			break
		}

		param, err := p.parseParam()
		if err != nil {
			return nil, err
		}

		params = append(params, param)

		if p.peek().IsToken() == TokenType_Comma {
			p.advance()
		} else {
			break
		}
	}

	return params, nil
}

// param ::= "mut"? IDENT ":" typeExpr
func (p *Parser) parseParam() (AstNode, error) {
	isMut := isKeyword(p.peek(), "mut")

	var start Position
	if isMut {
		tok := p.advance()
		start, _ = tok.Range()
	}

	ident, err := p.expect(TokenType_Ident)
	if err != nil {
		return nil, err
	}

	if !isMut {
		start, _ = ident.Range()
	}

	if _, err = p.expect(TokenType_Colon); err != nil {
		return nil, err
	}

	t, err := p.parseTypeExpr()
	if err != nil {
		return nil, err
	}

	_, end := t.Range()

	return &Parameter{
		Start: start,
		End:   end,
		Name:  ident.(*ValueToken).Value,
		Type:  t,
		Mut:   isMut,
	}, nil
}

// typeExpr ::= IDENT ( "[" "]" )* "?"?
func (p *Parser) parseTypeExpr() (AstNode, error) {
	token, err := p.expect(TokenType_Ident)
	if err != nil {
		return nil, err
	}
	start, end := token.Range()

	t := &Type{
		Start: start,
		End:   end,
		Name:  token.(*ValueToken).Value,
	}

	for p.peek().IsToken() == TokenType_BracketSquareOpen && p.peekN(1).IsToken() == TokenType_BracketSquareClose {
		p.advance()
		last := p.advance()
		t.IsArray = true
		_, t.End = last.Range()
	}

	if p.peek().IsToken() == TokenType_Question {
		last := p.advance()
		t.IsNullable = true
		_, t.End = last.Range()
	}

	return t, nil
}

// fnDecl ::= "fn" IDENT "(" params? ")" (":" typeExpr)? block
func (p *Parser) parseFnDecl() (AstNode, error) {
	start, _ := p.advance().Range()
	name, err := p.expect(TokenType_Ident)
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(TokenType_BracketParamOpen); err != nil {
		return nil, err
	}

	params, err := p.parseParams()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(TokenType_BracketParamClose); err != nil {
		return nil, err
	}

	var returnType AstNode
	if p.peek().IsToken() == TokenType_Colon {
		p.advance()
		returnType, err = p.parseTypeExpr()
		if err != nil {
			return nil, err
		}
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	_, end := body.Range()

	return &FunctionDeclaration{
		Start:      start,
		End:        end,
		Name:       name.(*ValueToken).Value,
		Params:     params,
		ReturnType: returnType,
		Body:       body,
	}, nil
}

// importStmt ::= "import" IDENT "from" STRING ";"
func (p *Parser) parseImport() (AstNode, error) {
	imp := p.advance()
	start, _ := imp.Range()

	imports := make([]string, 0)

	for {
		if p.peek().IsToken() != TokenType_Ident {
			return nil, fmt.Errorf("was expecting a ident")
		}

		ident := p.advance()

		imports = append(imports, ident.(*ValueToken).Value)

		if p.peek().IsToken() == TokenType_Comma {
			p.advance()
		} else {
			break
		}
	}

	if !isKeyword(p.peek(), "from") {
		return nil, fmt.Errorf("was expecting keyword 'from'")
	}
	p.advance()

	location, err := p.expect(TokenType_String)
	if err != nil {
		return nil, err
	}

	tok, err := p.expect(TokenType_Semicolon)
	if err != nil {
		return nil, err
	}

	_, end := tok.Range()

	return &ImportStatement{
		Start:   start,
		End:     end,
		Source:  location.(*ValueToken).Value,
		Imports: imports,
	}, nil
}

// whileStmt ::= "while" "(" expression ")" block
func (p *Parser) parseWhile() (AstNode, error) {
	keyword := p.advance()
	start, _ := keyword.Range()

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	block, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	_, end := block.Range()

	return &WhileStatement{
		Start:     start,
		End:       end,
		Condition: expr,
		Body:      block,
	}, nil
}

// structDecl ::= "struct" IDENT "{" ( param ";" )* "}"
func (p *Parser) parseStruct() (AstNode, error) {
	keyword := p.advance()
	start, _ := keyword.Range()

	ident, err := p.expect(TokenType_Ident)
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(TokenType_BracketCurlyOpen); err != nil {
		return nil, err
	}

	fields := make([]AstNode, 0)

	for {
		if p.peek().IsToken() != TokenType_Ident {
			break
		}

		param, err := p.parseParam()
		if err != nil {
			return nil, err
		}

		fields = append(fields, param)

		if _, err := p.expect(TokenType_Semicolon); err != nil {
			return nil, err
		}
	}

	tok, err := p.expect(TokenType_BracketCulryClose)
	if err != nil {
		return nil, err
	}
	_, end := tok.Range()

	return &StructStatement{
		Start:  start,
		End:    end,
		Name:   ident.(*ValueToken).Value,
		Fields: fields,
	}, nil
}

// varDecl ::= "mut"? IDENT ":" typeExpr "=" expression ";"
func (p *Parser) parseVarDecl() (AstNode, error) {
	keyword := p.advance()
	start, _ := keyword.Range()

	isMut := isKeyword(p.peek(), "mut")
	if isMut {
		p.advance()
	}

	ident, err := p.expect(TokenType_Ident)
	if err != nil {
		return nil, err
	}

	var varType AstNode
	if p.peek().IsToken() == TokenType_Colon {
		if _, err := p.expect(TokenType_Colon); err != nil {
			return nil, err
		}

		varType, err = p.parseTypeExpr()
		if err != nil {
			return nil, err
		}
	}

	if _, err := p.expect(TokenType_Equal); err != nil {
		return nil, err
	}

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	tok, err := p.expect(TokenType_Semicolon)
	if err != nil {
		return nil, err
	}
	_, end := tok.Range()

	return &VariableDeclaration{
		Start: start,
		End:   end,
		Type:  varType,
		Name:  ident.(*ValueToken).Value,
		Init:  expr,
	}, nil
}

// structImpl ::= "impl" IDENT "{" fnDecl* "}"
func (p *Parser) parseStructImpl() (AstNode, error) {
	keyword := p.advance()
	start, _ := keyword.Range()

	ident, err := p.expect(TokenType_Ident)
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(TokenType_BracketCurlyOpen); err != nil {
		return nil, err
	}

	methods := make([]AstNode, 0)
	for {
		next := p.peek()
		if next.IsToken() == TokenType_BracketCulryClose || next.IsToken() == TokenType_EOF {
			break
		}

		method, err := p.parseFnDecl()
		if err != nil {
			return nil, err
		}

		methods = append(methods, method)
	}

	tok, err := p.expect(TokenType_BracketCulryClose)
	if err != nil {
		return nil, err
	}
	_, end := tok.Range()

	return &StructImplStatement{
		Start:   start,
		End:     end,
		Name:    ident.(*ValueToken).Value,
		Methods: methods,
	}, nil
}

// returnStatement ::= "return" ( exprStmt | ";" )
func (p *Parser) parseReturnStatement() (AstNode, error) {
	start, end := p.advance().Range()

	var expr AstNode
	if p.peek().IsToken() != TokenType_Semicolon {
		stmt, err := p.parseExprStmt()
		if err != nil {
			return nil, err
		}

		expr = stmt
		_, end = expr.Range()
	} else {
		_, end = p.advance().Range()
	}

	return &ReturnStatement{
		Start: start,
		End:   end,
		Value: expr,
	}, nil
}

func (p *Parser) parseBreakStatement() (AstNode, error) {
	start, _ := p.advance().Range()

	tok, err := p.expect(TokenType_Semicolon)
	if err != nil {
		return nil, err
	}

	_, end := tok.Range()

	return &BreakStatement{
		Start: start,
		End:   end,
	}, nil
}

//#endregion
