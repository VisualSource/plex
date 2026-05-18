package parser

import (
	"errors"
	"io"
	"slices"

	"github.com/VisualSource/plex/internal/css/cssom"
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
func (p *CssParser) ParseAccordingToCssGrammar(stream io.Reader) ([]cssom.Component, error) {
	result, err := p.ParseListOfComponentValues(stream)
	if err != nil {
		return nil, err
	}
	// TODO: match against grammar

	return result, nil
}

// @see https://www.w3.org/TR/css-syntax-3/#parse-comma-list
func (p *CssParser) ParseListAccordingToCssGrammar(stream io.Reader) ([]any, error) {
	result := make([]any, 0)

	// white space
	//    -> result

	_list, err := p.ParseCommaListOfComponentValues(stream)
	if err != nil {
		return nil, err
	}

	//TODO: parse item with grammar

	return result, nil
}

// Intended to be the normal parser entry point, for parsing stylesheets.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-stylesheet
func (p *CssParser) ParseStylesheet(input io.Reader, location utils.StringOption) (*cssom.Stylesheet, error) {
	p.tok = tokenizer.NewCssTokenizer(input) // TODO: look into using shared tokenizer
	stylesheet := cssom.NewStylesheet(location)

	rules, err := p.consumeListOfRules(true)
	if err != nil {
		return nil, err
	}

	stylesheet.Value = rules

	return stylesheet, nil
}

// Intended for the content of at-rules such as @media. It differs from "Parse a stylesheet" in the handling of <CDO-token> and <CDC-token>.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-list-of-rules
func (p *CssParser) ParseListOfRules(input io.Reader) ([]cssom.Rule, error) {
	p.tok = tokenizer.NewCssTokenizer(input)

	rules, err := p.consumeListOfRules(false)
	if err != nil {
		return nil, err
	}

	return rules, nil
}

// Intended for use by the CSSStyleSheet#insertRule method, and similar functions which might exist, which parse text into a single rule.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-rule
func (p *CssParser) ParseRule(stream io.Reader) (*cssom.Rule, error) {
	p.tok = tokenizer.NewCssTokenizer(stream)

	var rule *cssom.Rule
	isFirstStage := true
	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		if token.IsToken() == tokenizer.TokenId_Whitespace {
			continue
		}

		if isFirstStage {
			switch token.IsToken() {
			case tokenizer.TokenId_EOF:
				return nil, errors.New("syntax error")
			case tokenizer.TokenId_AtKeyword:
				r, err := p.consumeQualifiedRule()
				if err != nil {
					return nil, err
				}

				if r == nil {
					return nil, errors.New("syntax error")
				}

				rule = r
				isFirstStage = false
				continue
			}
		}

		if token.IsToken() == tokenizer.TokenId_EOF {
			return rule, nil
		}

		return nil, errors.New("syntax error")
	}
}

// Used in @supports conditions. [CSS3-CONDITIONAL]
//
// Unlike "Parse a list of declarations", this parses only a declaration and not an at-rule.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-declaration
func (p *CssParser) ParseDeclaration(stream io.Reader) (*cssom.Declaration, error) {
	tok := tokenizer.NewCssTokenizer(stream)

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		if token.IsToken() == tokenizer.TokenId_Whitespace {
			continue
		}

		if token.IsToken() != tokenizer.TokenId_Ident {
			return nil, errors.New("syntax error")
		}

		result := p.consumeDeclaration(tok, token)
		if result != nil {
			return result, nil
		}

		return nil, errors.New("syntax error")
	}
}

// This algorithm parses the contents of style rules, which need to allow nested style rules and other at-rules.
// If you don’t need nested style rules, such as in @page or in @keyframes child rules, use parse a list of declarations.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-style-blocks-contents
func (p *CssParser) ParseStyleBlockContents(stream io.Reader) []*cssom.Declaration {
	tok := tokenizer.NewCssTokenizer(stream)

	return p.consumeStyleBlockContents(tok)
}

// For the contents of a style attribute, which parses text into the contents of a single style rule.

// Despite the name, this actually parses a mixed list of declarations and at-rules, as CSS 2.1 does for @page. Unexpected at-rules (which could be all of them, in a given context) are invalid and will be ignored by the consumer.
//
// This algorithm does not handle nested style rules. If your use requires that, use parse a style block’s contents.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-list-of-declarations
func (p *CssParser) ParseListOfDeclarations(stream io.Reader) []*cssom.Declaration {
	tok := tokenizer.NewCssTokenizer(stream)

	return p.consumeListOfDeclarations(tok)
}

// For things that need to consume a single value, like the parsing rules for attr().
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-component-value
func (p *CssParser) ParseComponentValue(stream io.Reader) (cssom.Component, error) {
	p.tok = tokenizer.NewCssTokenizer(stream)

	for {
		token, err := p.consumeToken()
		if err != nil {
			return cssom.Component{}, err
		}

		if token.IsToken() == tokenizer.TokenId_Whitespace {
			continue
		} else if token.IsToken() == tokenizer.TokenId_EOF {
			return cssom.Component{}, errors.New("syntax error")
		}

		p.reconsumeToken(token)
		break
	}

	value, err := p.consumeComponentValue()
	if err != nil {
		return cssom.Component{}, err
	}

	for {
		token, err := p.consumeToken()
		if err != nil {
			return cssom.Component{}, err
		}

		if token.IsToken() == tokenizer.TokenId_Whitespace {
			continue
		} else if token.IsToken() == tokenizer.TokenId_EOF {
			return value, nil
		}

		return cssom.Component{}, errors.New("syntax error")
	}

}

// for the contents of presentational attributes, which parse text into a single declaration’s value,
// or for parsing a stand-alone selector [SELECT](https://www.w3.org/TR/css-syntax-3/#biblio-select) or list of Media Queries [MEDIAQ](https://www.w3.org/TR/css-syntax-3/#biblio-mediaq),
// as in Selectors API or the media HTML attribute.
//
// https://www.w3.org/TR/css-syntax-3/#parse-list-of-component-values
func (p *CssParser) ParseListOfComponentValues(stream io.Reader) ([]cssom.Component, error) {
	p.tok = tokenizer.NewCssTokenizer(stream)

	list := make([]cssom.Component, 0)

	for {
		token, err := p.consumeToken()
		if err != nil {
			return nil, err
		}

		if token.IsToken() == tokenizer.TokenId_EOF {
			break
		}
		p.reconsumeToken(token)

		value, err := p.consumeComponentValue()
		if err != nil {
			return nil, err
		}
		list = append(list, value)

	}
	return list, nil
}

// https://www.w3.org/TR/css-syntax-3/#parse-comma-separated-list-of-component-values
func (p *CssParser) ParseCommaListOfComponentValues(stream io.Reader) ([][]cssom.Component, error) {
	p.tok = tokenizer.NewCssTokenizer(stream)
	clvs := make([][]cssom.Component, 0)

	for {
		sublist := make([]cssom.Component, 0)
		eof := false
		for {
			token, err := p.consumeToken()
			if err != nil {
				return nil, err
			}
			if token.IsToken() == tokenizer.TokenId_EOF {
				eof = true
				break
			}
			if token.IsToken() == tokenizer.TokenId_Comma {
				break
			}
			p.reconsumeToken(token)
			component, err := p.consumeComponentValue()
			if err != nil {
				return nil, err
			}
			sublist = append(sublist, component)
		}
		clvs = append(clvs, sublist) // single append point for both exit conditions
		if eof {
			break
		}
	}
	return clvs, nil
}

//#endregion

// #region algorithms
// https://www.w3.org/TR/css-syntax-3/#consume-list-of-rules
func (p *CssParser) consumeListOfRules(topLevel bool) ([]cssom.Rule, error) {
	rules := make([]cssom.Rule, 0)

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

			atRule := p.consumeAtRule()
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
				if !errors.Is(err, NoValue) {
					return nil, err
				}
				continue
			}
			rules = append(rules, value)

		}
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-at-rule
func (p *CssParser) consumeAtRule() cssom.Rule {
	return cssom.Rule{}
}

var NoValue = errors.New("no value")

// https://www.w3.org/TR/css-syntax-3/#consume-qualified-rule
func (p *CssParser) consumeQualifiedRule() (*cssom.Rule, error) {

	return nil, nil
}

// https://www.w3.org/TR/css-syntax-3/#consume-style-block
func (p *CssParser) consumeStyleBlockContents(tok *tokenizer.CssTokenizer) []*cssom.Declaration {
	decls := make([]*cssom.Declaration, 0)

	return decls
}

// https://www.w3.org/TR/css-syntax-3/#consume-list-of-declarations
func (p *CssParser) consumeListOfDeclarations(tok *tokenizer.CssTokenizer) []*cssom.Declaration {
	decls := make([]*cssom.Declaration, 0)

	return decls
}

// https://www.w3.org/TR/css-syntax-3/#consume-declaration
func (p *CssParser) consumeDeclaration(tok *tokenizer.CssTokenizer, ident tokenizer.Token) *cssom.Declaration {

	return nil
}

// https://www.w3.org/TR/css-syntax-3/#consume-component-value
func (p *CssParser) consumeComponentValue() (cssom.Component, error) {
	token, err := p.consumeToken()
	if err != nil {
		return cssom.Component{}, err
	}

	switch token.IsToken() {
	case tokenizer.TokenId_BracketSquareOpen, tokenizer.TokenId_BracketCurlyOpen, tokenizer.TokenId_BracketParamOpen:
		p.consumeSimpleBlock()
		return cssom.Component{}, nil
	case tokenizer.TokenId_Function:
		p.consumeFunction()
		return cssom.Component{}, nil
	default:
		return cssom.Component{}, nil
	}

}

// https://www.w3.org/TR/css-syntax-3/#consume-simple-block
func (p *CssParser) consumeSimpleBlock() {

}

// https://www.w3.org/TR/css-syntax-3/#consume-function
func (p *CssParser) consumeFunction() {}

//#endregion
