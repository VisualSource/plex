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
	p.tok = tokenizer.NewCssTokenizer(input, false) // TODO: look into using shared tokenizer
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
	p.tok = tokenizer.NewCssTokenizer(stream, false)
	return p.consumeStylesheetContents()
}

func (p *CssParser) ParseBlocksContents(stream io.Reader) {
	p.tok = tokenizer.NewCssTokenizer(stream, false)

	return p.consumeBlock()
}

//#endregion

// Intended for use by the CSSStyleSheet#insertRule method, and similar functions which might exist, which parse text into a single rule.
//
// @see https://drafts.csswg.org/css-syntax/#parse-rule
func (p *CssParser) ParseRule(stream io.Reader) (*Rule, error) {
	p.tok = tokenizer.NewCssTokenizer(stream, false)

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
	p.tok = tokenizer.NewCssTokenizer(stream, false)

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
	p.tok = tokenizer.NewCssTokenizer(stream, false)

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
	p.tok = tokenizer.NewCssTokenizer(stream, false)

	return p.consumeListOfComponentValues(nil)
}

// @see https://www.w3.org/TR/css-syntax-3/#parse-comma-separated-list-of-component-values
func (p *CssParser) ParseCommaListOfComponentValues(stream io.Reader) ([][]tokenizer.Token, error) {
	p.tok = tokenizer.NewCssTokenizer(stream, false)

	groups := make([][]tokenizer.Token, 0)

	delim := tokenizer.NewDataToken(tokenizer.TokenId_Comma)
	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		if token.IsToken() == tokenizer.TokenId_EOF {
			break
		}

		result, err := p.consumeListOfComponentValues(delim)
		if err != nil {
			return nil, err
		}

		groups = append(groups, result)
	}

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

// @see https://www.w3.org/TR/css-syntax-3/#consume-at-rule
func (p *CssParser) consumeAtRule(nested bool) (*Rule, error) {
	nameToken, err := p.consumeToken()
	if err != nil {
		return nil, err
	}

	if nameToken.IsToken() != tokenizer.TokenId_AtKeyword {
		return nil, errors.New("was expecting a at keyword token")
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
		case tokenizer.TokenId_Semicolon, tokenizer.TokenId_EOF:
			// invalid check?
			return atRule, nil
		case tokenizer.TokenId_BracketCurlyClose:
			if nested {
				// is valid

				return atRule, nil
			}

			atRule.Prelude = append(atRule.Prelude, token)
		case tokenizer.TokenId_BracketCurlyOpen:
			block, err := p.consumeBlock()
			if err != nil {
				return nil, err
			}

			atRule.ChildRules = append(atRule.ChildRules, block)

			// valid check

			return atRule, nil
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
func (p *CssParser) consumeQualifiedRule(stop tokenizer.Token, nested bool) (*Rule, error) {
	qr := &Rule{}

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		if stop != nil && stop.IsToken() == token.IsToken() {
			//TODO: parse erro
			return nil, nil
		}

		switch {
		case token.IsToken() == tokenizer.TokenId_EOF:
			//TODO: parse error
			return nil, nil
		case token.IsToken() == tokenizer.TokenId_BracketCurlyClose:
			//TODO: parse error
			if nested {
				return nil, nil
			}

			qr.Prelude = append(qr.Prelude, token)

		case token.IsToken() == tokenizer.TokenId_BracketCurlyOpen:

			//TODO

			return qr, nil
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

// https://drafts.csswg.org/css-syntax/#consume-block
func (p *CssParser) consumeBlock() ([]tokenizer.Token, error) {
	token, err := p.consumeToken()
	if err != nil {
		return nil, err
	}

	if token.IsToken() != tokenizer.TokenId_BracketCurlyOpen {
		return nil, errors.New("was expecting a '{' token")
	}

	rules, err := p.consumeBlocksContents()
	if err != nil {
		return nil, err
	}

	return rules, nil
}

// https://drafts.csswg.org/css-syntax/#consume-block-contents
func (p *CssParser) consumeBlocksContents() ([]tokenizer.Token, error) {
	rules := make([]tokenizer.Token, 0)
	decls := make([]tokenizer.Token, 0)

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		switch token.IsToken() {
		case tokenizer.TokenId_Whitespace, tokenizer.TokenId_Semicolon:
			continue
		case tokenizer.TokenId_EOF, tokenizer.TokenId_BracketCurlyClose:
			return rules, nil
		case tokenizer.TokenId_AtKeyword:
			decls := make([]tokenizer.Token, 0)

			rule, err := p.consumeAtRule(false)
			if err != nil {
				return nil, err
			}

			if rule != nil {
				rules = append(rules, rule)
			}
		default:
			//TODO: mark

			decl, err := p.consumeDeclaration(true)
			if err != nil {
				return nil, err
			}

			if decl != nil {
				decls = append(decls, decl)
				//discard mark

				continue
			}

			// resstore mark
			qr, err := p.consumeQualifiedRule(tokenizer.NewDataToken(tokenizer.TokenId_Semicolon), true)
			if err != nil {
				if errors.Is(err, ErrInvalidRule) {
					if len(decls) != 0 {

					}
					continue
				}
				return nil, err
			}

			if qr != nil {
				if len(decls) != 0 {

				}

			}

		}

	}

}

// Note: This algorithm assumes that the next input token has already been checked to be an <ident-token>.
//
// @see https://drafts.csswg.org/css-syntax/#consume-declaration
func (p *CssParser) consumeDeclaration(nested bool) (*Declaration, error) {
	token, err := p.consumeToken()
	if err != nil {
		return nil, err
	}

	if token.IsToken() != tokenizer.TokenId_Ident {
		p.reconsumeToken(token)
		err = p.consumeRemnantsOfBadDeclaration(nested)
		return nil, err
	}

	decl := &Declaration{
		Name: token,
	}

	if err := p.discardWhitespace(); err != nil {
		return nil, err
	}

	token, err = p.consumeToken()
	if err != nil {
		return nil, err
	}

	if token.IsToken() != tokenizer.TokenId_Colon {
		p.reconsumeToken(token)
		return nil, p.consumeRemnantsOfBadDeclaration(nested)
	}

	if err := p.discardWhitespace(); err != nil {
		return nil, err
	}

	decls, err := p.consumeListOfComponentValues(utils.Some(tokenizer.TokenId_Semicolon), nested)
	if err != nil {
		return nil, err
	}

	decl.Value = decls

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

	if lastIdx != -1 && secondLastIdx != -1 && isDelim(decl.Value[secondLastIdx], '!') && isIdent(decl.Value[lastIdx], "important", true) {
		decl.Value = slices.Delete(decl.Value, secondLastIdx, lastIdx+1)
		decl.Important = true
	}

	for len(decl.Value) > 0 && decl.Value[len(decl.Value)-1].IsToken() == tokenizer.TokenId_Whitespace {
		decl.Value = slices.Delete(decl.Value, len(decl.Value)-1, len(decl.Value))
	}

	switch {
	case isCustomPropertyName(decl.Name):
	case slices.ContainsFunc(decl.Value, isSimpleBlockWithCurlyOpen):
		seenCurly := false
		for _, v := range decl.Value {
			if v.IsToken() == tokenizer.TokenId_Whitespace {
				continue
			}
			if !seenCurly && isSimpleBlockWithCurlyOpen(v) {
				seenCurly = true
				continue
			}
			return nil, nil
		}
	case isIdent(decl.Name, "unicode-range", true):

	}

	if !p.declarationIsValid(decl) {
		return nil, nil
	}

	return decl, nil
}

// declarationIsValid implements the spec's "valid in the current context" check.
// Validity is defined per-property by individual CSS specs; until a property
// grammar / descriptor registry exists in this package, every declaration is
// treated as valid.
func (p *CssParser) declarationIsValid(decl *Declaration) bool {
	return true
}

// https://drafts.csswg.org/css-syntax/#consume-the-remnants-of-a-bad-declaration
func (p *CssParser) consumeRemnantsOfBadDeclaration(nested bool) error {
	for {
		token, err := p.consumeToken()
		if err != nil {
			return err
		}

		switch token.IsToken() {
		case tokenizer.TokenId_EOF, tokenizer.TokenId_Semicolon:
			return nil
		case tokenizer.TokenId_BracketCurlyClose:
			if nested {
				p.reconsumeToken(token)
				return nil
			}
		default:
			p.reconsumeToken(token)
			if _, err := p.consumeComponentValue(); err != nil {
				return err
			}
		}
	}
}

// @see https://drafts.csswg.org/css-syntax/#consume-list-of-components
func (p *CssParser) consumeListOfComponentValues(stop utils.Option[tokenizer.TokenId], nested bool) ([]tokenizer.Token, error) {
	values := make([]tokenizer.Token, 0)

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		if stop.IsSome() && stop.Is(token.IsToken()) {
			return values, nil
		}

		switch token.IsToken() {
		case tokenizer.TokenId_EOF:
			return values, nil
		case tokenizer.TokenId_BracketCurlyClose:
			if nested {
				return values, nil
			}

			//TODO: parse error
			values = append(values, token)
		default:
			p.reconsumeToken(token)
			value, err := p.consumeComponentValue()
			if err != nil {
				return nil, err
			}

			values = append(values, value)
		}
	}

}

// @see https://drafts.csswg.org/css-syntax/#consume-component-value
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

// https://drafts.csswg.org/css-syntax/#consume-simple-block
func (p *CssParser) consumeSimpleBlock() (*SimpleBlock, error) {
	startToken, err := p.consumeToken()
	if err != nil {
		return nil, err
	}

	switch startToken.IsToken() {
	case tokenizer.TokenId_BracketCurlyOpen, tokenizer.TokenId_BracketSquareOpen, tokenizer.TokenId_BracketParamOpen:
		break
	default:
		return nil, errors.New("was expecting a '{','(', or '[' token")
	}

	endToken := bracketMap[startToken.IsToken()]

	block := &SimpleBlock{
		StartDelim: startToken,
	}

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		if token.IsToken() == tokenizer.TokenId_EOF || token.IsToken() == endToken {
			return block, nil
		}

		p.reconsumeToken(token)
		value, err := p.consumeComponentValue()
		if err != nil {
			return nil, err
		}

		block.Value = append(block.Value, value)
	}

}

// Note: This algorithm assumes that the current input token has already been checked to be a <function-token>.
//
// @see https://drafts.csswg.org/css-syntax/#consume-function
func (p *CssParser) consumeFunction() (*Function, error) {
	fnT, err := p.consumeToken()
	if err != nil {
		return nil, err
	}

	if fnT.IsToken() != tokenizer.TokenId_FunctionToken {
		return nil, errors.New("was expecting a function token")
	}

	fn := &Function{
		Name: getTokenValueAsString(fnT),
	}

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		switch token.IsToken() {
		case tokenizer.TokenId_BracketParamClose, tokenizer.TokenId_EOF:
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
func (p *CssParser) consumeUnicodeRangeValue(input string) ([]tokenizer.Token, error) {
	prev := p.tok
	p.tok = tokenizer.NewCssTokenizer(strings.NewReader(input), true)
	defer func() { p.tok = prev }()
	return p.consumeListOfComponentValues(utils.None[tokenizer.TokenId](), false)
}

//#endregion
