package parser

import (
	"errors"
	"io"
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/css/tokenizer"
	"github.com/VisualSource/plex/internal/utils"
)

type CssParser struct {
	tok    *tokenizer.CssTokenizer
	tokens []tokenizer.Token
}

func NewCssParser() *CssParser {
	return &CssParser{}
}

func (p *CssParser) isEmpty() (bool, error) {
	token, err := p.consumeToken()
	if err != nil {
		return false, err
	}

	if token.IsToken() == tokenizer.TokenId_EOF {
		return true, nil
	}

	p.reconsumeToken(token)
	return false, nil
}

func (p *CssParser) discardWhitespace() error {
	for {
		token, err := p.consumeToken()
		if err != nil {
			return err
		}

		if token.IsToken() == tokenizer.TokenId_Whitespace {
			continue
		}

		p.reconsumeToken(token)
		break
	}

	return nil
}

func (p *CssParser) consumeToken() (tokenizer.Token, error) {
	if len(p.tokens) != 0 {
		last := p.tokens[len(p.tokens)-1]
		p.tokens = slices.Delete(p.tokens, len(p.tokens)-1, len(p.tokens))
		return last, nil
	}

	return p.tok.ConsumeToken()
}
func (p *CssParser) reconsumeToken(token tokenizer.Token) {
	p.tokens = append(p.tokens, token)
}

// #region Entry Points

// This algorithm, and parse a comma-separated list according to a CSS grammar, are usually the only parsing algorithms other specs will want to call.
// The remaining parsing algorithms are meant mostly for [CSSOM] and related "explicitly constructing CSS structures" cases.
// Consult the CSSWG for guidance first if you think you need to use one of the other algorithms.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-grammar
func (p *CssParser) ParseAccordingToCssGrammarStream(stream io.Reader) ([]any, error) {
	return nil, nil
}

// @see https://www.w3.org/TR/css-syntax-3/#parse-comma-list
func (p *CssParser) ParseListAccordingToCssGrammarStream(stream io.Reader) ([]any, error) {
	return nil, nil
}

// Intended to be the normal parser entry point, for parsing stylesheets.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-stylesheet
func (p *CssParser) ParseStylesheet(input io.Reader, location utils.StringOption) (*Stylesheet, error) {
	p.tok = tokenizer.NewCssTokenizer(input) // TODO: look into using shared tokenizer
	stylesheet := &Stylesheet{
		Location: location,
	}

	rules, err := p.consumeStylesheetContents()
	if err != nil {
		return nil, err
	}

	stylesheet.Value = rules

	return stylesheet, nil
}

// https://drafts.csswg.org/css-syntax/#parse-stylesheet-contents
func (p *CssParser) ParseStylesheetContents(stream io.Reader) ([]*Rule, error) {
	p.tok = tokenizer.NewCssTokenizer(stream)
	return p.consumeStylesheetContents()
}

func (p *CssParser) ParseBlocksContents(stream io.Reader) {
	p.tok = tokenizer.NewCssTokenizer(stream)

	return p.consumeBlock()
}

//#endregion

// Intended for use by the CSSStyleSheet#insertRule method, and similar functions which might exist, which parse text into a single rule.
//
// @see https://drafts.csswg.org/css-syntax/#parse-rule
func (p *CssParser) ParseRule(stream io.Reader) (*Rule, error) {
	p.tok = tokenizer.NewCssTokenizer(stream)

	if err := p.discardWhitespace(); err != nil {
		return nil, err
	}

	var rule *Rule

	token, err := p.consumeToken()
	if err != nil {
		return nil, err
	}

	switch token.IsToken() {
	case tokenizer.TokenId_EOF:
		return nil, ErrSyntax
	case tokenizer.TokenId_AtKeyword:
		atRule, err := p.consumeAtRule()
		if err != nil {
			return nil, err
		}
		rule = atRule
	default:
		qRule, err := p.consumeQualifiedRule()
		if err != nil {
			return nil, err
		}

		if qRule == nil {
			return nil, ErrSyntax
		}
		rule = qRule
	}

	if err := p.discardWhitespace(); err != nil {
		return nil, err
	}

	token, err = p.consumeToken()
	if err != nil {
		return nil, err
	}

	if token.IsToken() == tokenizer.TokenId_EOF {
		return rule, nil
	}

	return nil, ErrSyntax
}

// Used in @supports conditions. [CSS3-CONDITIONAL]
//
// Unlike "Parse a list of declarations", this parses only a declaration and not an at-rule.
//
// @see https://drafts.csswg.org/css-syntax/#parse-declaration
func (p *CssParser) ParseDeclaration(stream io.Reader) (*Declaration, error) {
	p.tok = tokenizer.NewCssTokenizer(stream)

	if err := p.discardWhitespace(); err != nil {
		return nil, err
	}

	decl, err := p.consumeDeclaration()
	if err != nil {
		return nil, err
	}

	if decl != nil {
		return decl, nil
	}

	return nil, ErrSyntax
}

// For things that need to consume a single value, like the parsing rules for attr().
//
// @see https://drafts.csswg.org/css-syntax/#parse-component-value
func (p *CssParser) ParseComponentValue(stream io.Reader) (tokenizer.Token, error) {
	p.tok = tokenizer.NewCssTokenizer(stream)

	if err := p.discardWhitespace(); err != nil {
		return nil, err
	}

	empty, err := p.isEmpty()
	if err != nil {
		return nil, err
	}

	if empty {
		return nil, ErrSyntax
	}

	value, err := p.consumeComponentValue()
	if err != nil {
		return nil, err
	}

	if err := p.discardWhitespace(); err != nil {
		return nil, err
	}

	empty, err = p.isEmpty()
	if err != nil {
		return nil, err
	}

	if empty {
		return nil, ErrSyntax
	}

	return value, nil
}

// for the contents of presentational attributes, which parse text into a single declaration’s value,
// or for parsing a stand-alone selector [SELECT](https://www.w3.org/TR/css-syntax-3/#biblio-select) or list of Media Queries [MEDIAQ](https://www.w3.org/TR/css-syntax-3/#biblio-mediaq),
// as in Selectors API or the media HTML attribute.
//
// @see https://drafts.csswg.org/css-syntax/#parse-list-of-component-values
func (p *CssParser) ParseListOfComponentValues(stream io.Reader) ([]tokenizer.Token, error) {
	p.tok = tokenizer.NewCssTokenizer(stream)

}

// @see https://www.w3.org/TR/css-syntax-3/#parse-comma-separated-list-of-component-values
func (p *CssParser) ParseCommaListOfComponentValues(stream io.Reader) ([][]tokenizer.Token, error) {
	p.tok = tokenizer.NewCssTokenizer(stream)

	groups := make([][]tokenizer.Token, 0)

	//TODO

	return groups, nil
}

//#endregion

// #region algorithms

// @see https://drafts.csswg.org/css-syntax/#consume-stylesheet-contents
func (p *CssParser) consumeStylesheetContents() ([]*Rule, error) {
	rules := make([]*Rule, 0)

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		switch token.IsToken() {
		case tokenizer.TokenId_Whitespace, tokenizer.TokenId_CDC, tokenizer.TokenId_CDO:
			continue
		case tokenizer.TokenId_EOF:
			return rules, nil
		case tokenizer.TokenId_AtKeyword:
			atRule, err := p.consumeAtRule()
			if err != nil {
				return nil, err
			}

			if atRule != nil {
				rules = append(rules, atRule)
			}
		default:
			rule, err := p.consumeQualifiedRule()
			if err != nil {
				return nil, err
			}

			if rule != nil {
				rules = append(rules, rule)
			}
		}
	}
}

// @see https://www.w3.org/TR/css-syntax-3/#consume-list-of-rules
func (p *CssParser) consumeListOfRules(topLevel bool) ([]*Rule, error) {
	rules := make([]*Rule, 0)

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		switch token.IsToken() {
		case tokenizer.TokenId_Whitespace:
			continue
		case tokenizer.TokenId_EOF:
			return rules, nil
		case tokenizer.TokenId_AtKeyword:
			p.reconsumeToken(token)

			atRule, err := p.consumeAtRule()
			if err != nil {
				return nil, err
			}
			rules = append(rules, atRule)
		case tokenizer.TokenId_CDO, tokenizer.TokenId_CDC:
			if topLevel {
				continue
			}
			fallthrough
		default:
			p.reconsumeToken(token)

			value, err := p.consumeQualifiedRule()
			if err != nil {
				return nil, err
			}

			if value != nil {
				rules = append(rules, value)
			}
		}
	}
}

// @see https://www.w3.org/TR/css-syntax-3/#consume-at-rule
func (p *CssParser) consumeAtRule() (*Rule, error) {
	nameToken, err := p.consumeToken()
	if err != nil {
		return nil, err
	}

	atRule := &Rule{
		Name: nameToken,
	}

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		switch token.IsToken() {
		case tokenizer.TokenId_Semicolon:
			return atRule, nil
		case tokenizer.TokenId_EOF:
			//TOOD: parse error
			return atRule, nil
		case tokenizer.TokenId_BracketCurlyOpen:
			block, err := p.consumeSimpleBlock()
			if err != nil {
				return nil, err
			}

			atRule.Value = block

			return atRule, nil
		case TokenId_SimpleBlock:
			if block, ok := token.(*SimpleBlock); ok && block.StartDelim.IsToken() == tokenizer.TokenId_BracketCurlyOpen {
				atRule.Value = block
				return atRule, nil
			}
			fallthrough
		default:
			p.reconsumeToken(token)
			comp, err := p.consumeComponentValue()
			if err != nil {
				return nil, err
			}

			atRule.Prelude = append(atRule.Prelude, comp)
		}
	}
}

// @see https://www.w3.org/TR/css-syntax-3/#consume-qualified-rule
func (p *CssParser) consumeQualifiedRule() (*Rule, error) {
	qr := &Rule{}

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		switch {
		case token.IsToken() == tokenizer.TokenId_EOF:
			//TODO: parse error
			return nil, nil
		case token.IsToken() == tokenizer.TokenId_BracketCurlyOpen:
			block, err := p.consumeSimpleBlock()
			if err != nil {
				return nil, err
			}

			qr.Value = block

			return qr, nil
		case token.IsToken() == TokenId_SimpleBlock:
			if tag, ok := token.(*SimpleBlock); ok && tag.StartDelim.IsToken() == tokenizer.TokenId_BracketCurlyOpen {
				qr.Value = tag
				return qr, nil
			}
			fallthrough
		default:
			p.reconsumeToken(token)

			value, err := p.consumeComponentValue()
			if err != nil {
				return nil, err
			}

			qr.Prelude = append(qr.Prelude, value)
		}
	}
}

// @see https://www.w3.org/TR/css-syntax-3/#consume-style-block
func (p *CssParser) consumeStyleBlockContents(tok *tokenizer.CssTokenizer) ([]tokenizer.Token, error) {
	decls := make([]tokenizer.Token, 0)

	rules := make([]tokenizer.Token, 0)
	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		switch token.IsToken() {
		case tokenizer.TokenId_Whitespace, tokenizer.TokenId_Semicolon:
			continue
		case tokenizer.TokenId_EOF:
			decls = append(decls, rules...)
			return decls, nil
		case tokenizer.TokenId_AtKeyword:
			p.reconsumeToken(token)
			atRule, err := p.consumeAtRule()
			if err != nil {
				return nil, err
			}
			decls = append(decls, atRule)
		case tokenizer.TokenId_Ident:
			temp := []tokenizer.Token{token}

			for {
				t, err := p.consumeToken()
				if err != nil {
					return nil, err
				}

				if t.IsToken() == tokenizer.TokenId_Semicolon || t.IsToken() == tokenizer.TokenId_EOF {
					p.reconsumeToken(t)
					break
				}

				p.reconsumeToken(t)
				value, err := p.consumeComponentValue()
				if err != nil {
					return nil, err
				}

				temp = append(temp, value)
			}

			// Consume a declaration from the temporary list: push EOF beneath
			// temp (so consumeDeclaration stops at the end of temp) then temp
			// in reverse so it pops in original order. The semicolon/EOF that
			// terminated the inner loop is already on the stack below, ready
			// for the outer loop.
			p.reconsumeToken(tokenizer.EOFToken{})
			for i := len(temp) - 1; i >= 0; i-- {
				p.reconsumeToken(temp[i])
			}

			dec, err := p.consumeDeclaration()
			if err != nil {
				return nil, err
			}
			if dec != nil {
				decls = append(decls, dec)
			}
		case tokenizer.TokenId_Delim:
			if tag, ok := token.(*tokenizer.SingleCharacterToken); ok && tag.Value == '&' {
				p.reconsumeToken(token)
				rule, err := p.consumeQualifiedRule()
				if err != nil {
					return nil, err
				}
				if rule != nil {
					rules = append(rules, rule)
				}
				continue
			}
			fallthrough
		default:
			//TODO: parse error
			p.reconsumeToken(token)

			for {
				t, err := p.consumeToken()
				if err != nil {
					return nil, err
				}

				if t.IsToken() == tokenizer.TokenId_Semicolon || t.IsToken() == tokenizer.TokenId_EOF {
					p.reconsumeToken(t)
					break
				}

				p.reconsumeToken(t)
				if _, err := p.consumeComponentValue(); err != nil {
					return nil, err
				}
			}
		}

	}
}

// @see https://www.w3.org/TR/css-syntax-3/#consume-list-of-declarations
func (p *CssParser) consumeListOfDeclarations(tok *tokenizer.CssTokenizer) ([]tokenizer.Token, error) {
	decls := make([]tokenizer.Token, 0)

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		switch token.IsToken() {
		case tokenizer.TokenId_Whitespace, tokenizer.TokenId_Semicolon:
			continue
		case tokenizer.TokenId_AtKeyword:
			p.reconsumeToken(token)
			atRule, err := p.consumeAtRule()
			if err != nil {
				return nil, err
			}

			decls = append(decls, atRule)

		case tokenizer.TokenId_EOF:
			return decls, nil
		case tokenizer.TokenId_Ident:
			temp := []tokenizer.Token{token}

			for {
				t, err := p.consumeToken()
				if err != nil {
					return nil, err
				}

				if t.IsToken() == tokenizer.TokenId_EOF || t.IsToken() == tokenizer.TokenId_Semicolon {
					p.reconsumeToken(t)
					break
				}

				p.reconsumeToken(t)
				value, err := p.consumeComponentValue()
				if err != nil {
					return nil, err
				}

				temp = append(temp, value)
			}

			// Consume a declaration from the temporary list: push EOF beneath temp
			// (so consumeDeclaration stops at the end of temp) then temp in reverse
			// so it pops in original order. The semicolon/EOF that terminated the
			// inner loop is already on the stack below, ready for the outer loop.
			p.reconsumeToken(tokenizer.EOFToken{})
			for i := len(temp) - 1; i >= 0; i-- {
				p.reconsumeToken(temp[i])
			}

			dec, err := p.consumeDeclaration()
			if err != nil {
				return nil, err
			}
			if dec != nil {
				decls = append(decls, dec)
			}
		default:
			//TODO: parse error
			p.reconsumeToken(token)

			for {
				t, err := p.consumeToken()
				if err != nil {
					return nil, err
				}

				if t.IsToken() == tokenizer.TokenId_Semicolon || t.IsToken() == tokenizer.TokenId_EOF {
					p.reconsumeToken(t)
					break
				}

				p.reconsumeToken(token)
				if _, err = p.consumeComponentValue(); err != nil {
					return nil, err
				}
			}
		}
	}
}

// Note: This algorithm assumes that the next input token has already been checked to be an <ident-token>.
//
// @see https://www.w3.org/TR/css-syntax-3/#consume-declaration
func (p *CssParser) consumeDeclaration() (*Declaration, error) {
	nameToken, err := p.consumeToken()
	if err != nil {
		return nil, err
	}

	decl := &Declaration{
		Name: nameToken,
	}

	firstPass := true
	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		if token.IsToken() == tokenizer.TokenId_Whitespace {
			continue
		}

		if firstPass {
			switch token.IsToken() {
			case tokenizer.TokenId_Colon:
				firstPass = false
			default:
				//TODO: parse error
				return nil, nil
			}

			continue
		}

		if token.IsToken() == tokenizer.TokenId_EOF {
			break
		}

		p.reconsumeToken(token)

		value, err := p.consumeComponentValue()
		if err != nil {
			return nil, err
		}

		decl.Value = append(decl.Value, value)
	}

	lastIdx, secondLastIdx := -1, -1
	for i := len(decl.Value) - 1; i >= 0; i-- {
		if decl.Value[i].IsToken() == tokenizer.TokenId_Whitespace {
			continue
		}
		if lastIdx == -1 {
			lastIdx = i
		} else {
			secondLastIdx = i
			break
		}
	}

	if lastIdx != -1 && secondLastIdx != -1 {
		bang, bangOk := decl.Value[secondLastIdx].(*tokenizer.SingleCharacterToken)
		ident, identOk := decl.Value[lastIdx].(*tokenizer.MultiCharacterToken)
		if bangOk && bang.Type == tokenizer.TokenId_Delim && bang.Value == '!' &&
			identOk && ident.Type == tokenizer.TokenId_Ident && strings.EqualFold(ident.Value, "important") {
			decl.Value = slices.Delete(decl.Value, lastIdx, lastIdx+1)
			decl.Value = slices.Delete(decl.Value, secondLastIdx, secondLastIdx+1)
			decl.Important = true
		}
	}

	for len(decl.Value) > 0 && decl.Value[len(decl.Value)-1].IsToken() == tokenizer.TokenId_Whitespace {
		decl.Value = decl.Value[:len(decl.Value)-1]
	}

	return decl, nil
}

// @see https://www.w3.org/TR/css-syntax-3/#consume-component-value
func (p *CssParser) consumeComponentValue() (tokenizer.Token, error) {
	token, err := p.consumeToken()
	if err != nil {
		return nil, err
	}

	switch token.IsToken() {
	case tokenizer.TokenId_BracketSquareOpen, tokenizer.TokenId_BracketCurlyOpen, tokenizer.TokenId_BracketParamOpen:
		p.reconsumeToken(token)
		return p.consumeSimpleBlock()
	case tokenizer.TokenId_FunctionToken:
		p.reconsumeToken(token)
		return p.consumeFunction()
	default:
		return token, nil
	}
}

var bracketMap map[tokenizer.TokenId]tokenizer.TokenId = map[tokenizer.TokenId]tokenizer.TokenId{
	tokenizer.TokenId_BracketCurlyOpen:  tokenizer.TokenId_BracketCurlyClose,
	tokenizer.TokenId_BracketParamOpen:  tokenizer.TokenId_BracketParamClose,
	tokenizer.TokenId_BracketSquareOpen: tokenizer.TokenId_BracketSquareClose,
}

// Note: This algorithm assumes that the current input token has already been checked to be an <{-token>, <[-token>, or <(-token>.
//
// Note: CSS has an unfortunate syntactic ambiguity between blocks that can contain declarations and blocks that can contain qualified rules,
// so any "consume" algorithms that handle rules will initially use this more generic algorithm rather than the more specific
// consume a list of declarations or consume a list of rules algorithms. These more specific algorithms are instead invoked when grammars are applied,
// depending on whether it contains a <declaration-list> or a <rule-list>/<stylesheet>.
//
// @see https://www.w3.org/TR/css-syntax-3/#consume-simple-block
func (p *CssParser) consumeSimpleBlock() (*SimpleBlock, error) {
	startingToken, err := p.consumeToken()
	if err != nil {
		return nil, err
	}

	endDelim, ok := bracketMap[startingToken.IsToken()]
	if !ok {
		return nil, errors.New("invalid starting token")
	}

	block := &SimpleBlock{
		StartDelim: startingToken,
	}

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		switch {
		case token.IsToken() == endDelim:
			return block, nil
		case token.IsToken() == tokenizer.TokenId_EOF:
			//TODO: parse error
			return block, nil
		default:
			p.reconsumeToken(token)

			value, err := p.consumeComponentValue()
			if err != nil {
				return nil, err
			}
			block.Value = append(block.Value, value)
		}
	}
}

// Note: This algorithm assumes that the current input token has already been checked to be a <function-token>.
//
// @see https://www.w3.org/TR/css-syntax-3/#consume-function
func (p *CssParser) consumeFunction() (*Function, error) {
	fnT, err := p.consumeToken()
	if err != nil {
		return nil, err
	}

	v, ok := fnT.(*tokenizer.MultiCharacterToken)
	if !ok {
		return nil, errors.New("was expecting a multi character token")
	}

	fn := &Function{
		Name: v.Value,
	}

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		switch {
		case token.IsToken() == tokenizer.TokenId_BracketParamClose:
			return fn, nil
		case token.IsToken() == tokenizer.TokenId_EOF:
			//TODO: parse error
			return fn, nil
		default:
			p.reconsumeToken(token)

			value, err := p.consumeComponentValue()
			if err != nil {
				return nil, err
			}
			fn.Value = append(fn.Value, value)
		}
	}
}

// @see https://drafts.csswg.org/css-syntax/#consume-unicode-range-value
func (p *CssParser) consumeUnicodeRangeValue() {

}

func (p *CssParser) consumeBlock()          {}
func (p *CssParser) consumeBlocksContents() {}

//#endregion
