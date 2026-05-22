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
	marks  [][]tokenizer.Token
}

func NewCssParser() *CssParser {
	return &CssParser{}
}

//#region helpers

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
	var (
		token tokenizer.Token
		err   error
	)
	if len(p.tokens) != 0 {
		token = p.tokens[len(p.tokens)-1]
		p.tokens = slices.Delete(p.tokens, len(p.tokens)-1, len(p.tokens))
	} else {
		token, err = p.tok.ConsumeToken()
		if err != nil {
			return nil, err
		}
	}

	if n := len(p.marks); n != 0 {
		p.marks[n-1] = append(p.marks[n-1], token)
	}
	return token, nil
}
func (p *CssParser) reconsumeToken(token tokenizer.Token) {
	if n := len(p.marks); n != 0 {
		buf := p.marks[n-1]
		if m := len(buf); m != 0 && buf[m-1] == token {
			p.marks[n-1] = buf[:m-1]
		}
	}
	p.tokens = append(p.tokens, token)
}

// mark pushes a new mark onto the mark stack. Every token returned by
// consumeToken while this mark is the top of stack is recorded; restoreMark
// pushes those tokens back so they reconsume in order, discardMark drops them.
// Marks may nest because consumeDeclaration performs its own reconsumption.
func (p *CssParser) mark() {
	p.marks = append(p.marks, nil)
}

func (p *CssParser) discardMark() {
	n := len(p.marks)
	if n == 0 {
		return
	}
	buf := p.marks[n-1]
	p.marks = p.marks[:n-1]
	// Tokens consumed under this mark stay consumed, but any outer mark must
	// still see them as part of its window.
	if outer := len(p.marks); outer != 0 {
		p.marks[outer-1] = append(p.marks[outer-1], buf...)
	}
}

func (p *CssParser) restoreMark() {
	n := len(p.marks)
	if n == 0 {
		return
	}
	buf := p.marks[n-1]
	p.marks = p.marks[:n-1]
	// Push recorded tokens back so consumeToken returns them in original order.
	for i := len(buf) - 1; i >= 0; i-- {
		p.tokens = append(p.tokens, buf[i])
	}
}

//#endregion

//#region Entry Points

// @see https://www.w3.org/TR/css-syntax-3/#parse-grammar
func (p *CssParser) ParseAccordingToCssGrammarStream(stream io.Reader) ([]any, error) {
	return nil, nil
}

// @see https://www.w3.org/TR/css-syntax-3/#parse-comma-list
func (p *CssParser) ParseListAccordingToCssGrammarStream(stream io.Reader) ([]any, error) {
	return nil, nil
}

// intended to be the normal parser entry point, for parsing stylesheets.
//
// @see https://drafts.csswg.org/css-syntax/#parse-stylesheet
func (p *CssParser) ParseStylesheet(input io.Reader, location utils.StringOption) (*Stylesheet, error) {
	p.tok = tokenizer.NewCssTokenizer(input, false) // TODO: look into using shared tokenizer
	stylesheet := &Stylesheet{
		Location: location,
	}

	rules, err := p.consumeStylesheetContents()
	if err != nil {
		return nil, err
	}

	stylesheet.Rules = rules

	return stylesheet, nil
}

// is intended for use by the CSSStyleSheet replace() method, and similar, which parse text into the contents of an existing stylesheet.
//
// https://drafts.csswg.org/css-syntax/#parse-stylesheet-contents
func (p *CssParser) ParseStylesheetContents(stream io.Reader) ([]*Rule, error) {
	p.tok = tokenizer.NewCssTokenizer(stream, false)
	return p.consumeStylesheetContents()
}

// is intended for parsing the contents of any block in CSS (including things like the style attribute),
// and APIs such as the CSSStyleDeclaration cssText attribute.
//
// https://drafts.csswg.org/css-syntax/#parse-block-contents
func (p *CssParser) ParseBlocksContents(stream io.Reader) ([]tokenizer.Token, error) {
	p.tok = tokenizer.NewCssTokenizer(stream, false)
	return p.consumeBlocksContents()
}

// is intended for use by the CSSStyleSheet insertRule() method, and similar, which parse text into a single rule.
// CSSStyleSheet#insertRule method, and similar functions which might exist, which parse text into a single rule.
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
		p.reconsumeToken(token)
		atRule, err := p.consumeAtRule(false)
		if err != nil {
			if errors.Is(err, ErrInvalidRule) {
				return nil, ErrSyntax
			}
			return nil, err
		}
		if atRule == nil {
			return nil, ErrSyntax
		}

		rule = atRule
	default:
		p.reconsumeToken(token)
		qRule, err := p.consumeQualifiedRule(nil, false)
		if err != nil {
			if errors.Is(err, ErrInvalidRule) {
				return nil, ErrSyntax
			}

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

// is used in @supports conditions. [CSS3-CONDITIONAL]
//
// @see https://drafts.csswg.org/css-syntax/#parse-declaration
func (p *CssParser) ParseDeclaration(stream io.Reader) (*Declaration, error) {
	p.tok = tokenizer.NewCssTokenizer(stream, false)

	if err := p.discardWhitespace(); err != nil {
		return nil, err
	}

	decl, err := p.consumeDeclaration(false)
	if err != nil {
		return nil, err
	}

	if decl != nil {
		return decl, nil
	}

	return nil, ErrSyntax
}

// is for things that need to consume a single value, like the parsing rules for attr().
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

	if !empty {
		return nil, ErrSyntax
	}

	return value, nil
}

// is for the contents of presentational attributes, which parse text into a single declaration’s value,
// or for parsing a stand-alone selector [SELECT] or list of Media Queries [MEDIAQ], as in Selectors API or the media HTML attribute.
//
// @see https://drafts.csswg.org/css-syntax/#parse-list-of-component-values
func (p *CssParser) ParseListOfComponentValues(stream io.Reader) ([]tokenizer.Token, error) {
	p.tok = tokenizer.NewCssTokenizer(stream, false)

	return p.consumeListOfComponentValues(utils.None[tokenizer.TokenId](), false)
}

// @see https://drafts.csswg.org/css-syntax/#parse-comma-separated-list-of-component-values
func (p *CssParser) ParseCommaListOfComponentValues(stream io.Reader) ([][]tokenizer.Token, error) {
	p.tok = tokenizer.NewCssTokenizer(stream, false)

	groups := make([][]tokenizer.Token, 0)

	delim := utils.Some(tokenizer.TokenId_Comma)
	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		if token.IsToken() == tokenizer.TokenId_EOF {
			break
		}

		p.reconsumeToken(token)
		result, err := p.consumeListOfComponentValues(delim, false)
		if err != nil {
			return nil, err
		}

		if _, err = p.consumeToken(); err != nil {
			return nil, err
		} // eat delim token

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
			p.reconsumeToken(token)

			atRule, err := p.consumeAtRule(false)
			if err != nil {
				return nil, err
			}

			if atRule != nil {
				rules = append(rules, atRule)
			}
		default:
			p.reconsumeToken(token)
			rule, err := p.consumeQualifiedRule(nil, false)
			if err != nil {
				return nil, err
			}

			if rule != nil {
				rules = append(rules, rule)
			}
		}
	}
}

// @see https://drafts.csswg.org/css-syntax/#consume-at-rule
func (p *CssParser) consumeAtRule(nested bool) (*Rule, error) {
	nameToken, err := p.consumeToken()
	if err != nil {
		return nil, err
	}

	if nameToken.IsToken() != tokenizer.TokenId_AtKeyword {
		return nil, errors.New("was expecting a at keyword token")
	}

	atRule := &Rule{
		Name: getTokenValueAsString(nameToken),
	}

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		switch token.IsToken() {
		case tokenizer.TokenId_Semicolon, tokenizer.TokenId_EOF:
			if !p.ruleIsValid(atRule) {
				return nil, nil
			}
			return atRule, nil
		case tokenizer.TokenId_BracketCurlyClose:
			if nested {
				p.reconsumeToken(token)
				if p.ruleIsValid(atRule) {
					return atRule, nil
				}
				return atRule, nil
			}

			atRule.Prelude = append(atRule.Prelude, token)
		case tokenizer.TokenId_BracketCurlyOpen:
			p.reconsumeToken(token)
			block, err := p.consumeBlock()
			if err != nil {
				return nil, err
			}

			atRule.ChildRules = block

			if !p.ruleIsValid(atRule) {
				return nil, nil
			}

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

// @see https://drafts.csswg.org/css-syntax/#consume-qualified-rule
func (p *CssParser) consumeQualifiedRule(stop tokenizer.Token, nested bool) (*Rule, error) {
	qr := &Rule{}

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		if stop != nil && stop.IsToken() == token.IsToken() {
			return nil, nil
		}

		switch {
		case token.IsToken() == tokenizer.TokenId_EOF:
			return nil, nil
		case token.IsToken() == tokenizer.TokenId_BracketCurlyClose:
			if nested {
				p.reconsumeToken(token)
				return nil, nil
			}

			qr.Prelude = append(qr.Prelude, token)

		case token.IsToken() == tokenizer.TokenId_BracketCurlyOpen:
			if preludeLooksLikeCustomPropertyDecl(qr.Prelude) {
				if nested {
					if err := p.consumeRemnantsOfBadDeclaration(true); err != nil {
						return nil, err
					}
					return nil, nil
				}
				p.reconsumeToken(token)
				if _, err := p.consumeBlock(); err != nil {
					return nil, err
				}
				return nil, nil
			}

			p.reconsumeToken(token)
			childRules, err := p.consumeBlock()
			if err != nil {
				return nil, err
			}

			if len(childRules) > 0 {
				if dl, ok := childRules[0].(*DeclarationList); ok {
					qr.Declarations = dl
					childRules = childRules[1:]
				}
			}
			for i, v := range childRules {
				if dl, ok := v.(*DeclarationList); ok {
					childRules[i] = &NestedDeclarations{Value: dl, Start: dl.Start, End: dl.End}
				}
			}
			qr.ChildRules = childRules

			if !p.ruleIsValid(qr) {
				return nil, ErrInvalidRule
			}
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

// @see https://drafts.csswg.org/css-syntax/#consume-block
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

	if _, err := p.consumeToken(); err != nil {
		return nil, err
	}

	return rules, nil
}

// @see https://drafts.csswg.org/css-syntax/#consume-block-contents
func (p *CssParser) consumeBlocksContents() ([]tokenizer.Token, error) {
	var rules []tokenizer.Token
	var decls []*Declaration

	flushDecls := func() {
		if len(decls) == 0 {
			return
		}
		rules = append(rules, &DeclarationList{Value: decls})
		decls = nil
	}

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		switch token.IsToken() {
		case tokenizer.TokenId_Whitespace, tokenizer.TokenId_Semicolon:
			continue
		case tokenizer.TokenId_EOF, tokenizer.TokenId_BracketCurlyClose:
			p.reconsumeToken(token)
			flushDecls()
			return rules, nil
		case tokenizer.TokenId_AtKeyword:
			p.reconsumeToken(token)
			flushDecls()

			rule, err := p.consumeAtRule(true)
			if err != nil {
				return nil, err
			}

			if rule != nil {
				rules = append(rules, rule)
			}
		default:
			p.reconsumeToken(token)
			p.mark()

			decl, err := p.consumeDeclaration(true)
			if err != nil {
				p.discardMark()
				return nil, err
			}

			if decl != nil {
				p.discardMark()
				decls = append(decls, decl)
				continue
			}

			p.restoreMark()

			qr, err := p.consumeQualifiedRule(tokenizer.NewDataToken(tokenizer.TokenId_Semicolon), true)
			switch {
			case errors.Is(err, ErrInvalidRule):
				flushDecls()
			case err != nil:
				return nil, err
			case qr == nil:
				// nothing returned — do nothing
			default:
				flushDecls()
				rules = append(rules, qr)
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
		decl.OriginalText = utils.Some(p.valueSourceSegment(decl))
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
		tokens, err := p.consumeUnicodeRangeValue(p.valueSourceSegment(decl))
		if err != nil {
			return nil, err
		}
		decl.Value = tokens
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

// qualifiedRuleIsValid implements the spec's "valid in the current context"
// check for qualified rules. Until a context-aware rule grammar exists in this
// package, every qualified rule is treated as valid.
func (p *CssParser) ruleIsValid(rule *Rule) bool {
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
			p.reconsumeToken(token)
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

	startPos, _ := startToken.Range()
	block := &SimpleBlock{
		StartDelim: startToken,
		Start:      startPos,
	}

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		if token.IsToken() == tokenizer.TokenId_EOF || token.IsToken() == endToken {
			_, block.End = token.Range()
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

	startPos, _ := fnT.Range()
	fn := &Function{
		Name:  getTokenValueAsString(fnT),
		Start: startPos,
	}

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		switch token.IsToken() {
		case tokenizer.TokenId_BracketParamClose, tokenizer.TokenId_EOF:
			_, fn.End = token.Range()
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
