package parser

import "github.com/VisualSource/plex/internal/css/cssom"

type CssParser struct{}

func NewCssParser() *CssParser {
	return &CssParser{}
}

// #region Entry Points

// This algorithm, and parse a comma-separated list according to a CSS grammar, are usually the only parsing algorithms other specs will want to call.
// The remaining parsing algorithms are meant mostly for [CSSOM] and related "explicitly constructing CSS structures" cases.
// Consult the CSSWG for guidance first if you think you need to use one of the other algorithms.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-grammar
func (p *CssParser) ParseAccordingToCssGrammar() error {
	return nil
}

// @see https://www.w3.org/TR/css-syntax-3/#parse-comma-list
func (p *CssParser) ParseListAccordingToCssGrammar() []any {
	return nil
}

// Intended to be the normal parser entry point, for parsing stylesheets.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-stylesheet
func (p *CssParser) ParseStylesheet() *cssom.Stylesheet {
	return cssom.NewStylesheet()
}

// Intended for the content of at-rules such as @media. It differs from "Parse a stylesheet" in the handling of <CDO-token> and <CDC-token>.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-list-of-rules
func (p *CssParser) ParseListOfRules() {}

// Intended for use by the CSSStyleSheet#insertRule method, and similar functions which might exist, which parse text into a single rule.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-rule
func (p *CssParser) ParseRule() (cssom.Rule, error) {
	return cssom.Rule{}, nil
}

// Used in @supports conditions. [CSS3-CONDITIONAL]
//
// Unlike "Parse a list of declarations", this parses only a declaration and not an at-rule.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-declaration
func (p *CssParser) ParseDeclaration() (any, error) {

	return nil, nil
}

// This algorithm parses the contents of style rules, which need to allow nested style rules and other at-rules.
// If you don’t need nested style rules, such as in @page or in @keyframes child rules, use parse a list of declarations.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-style-blocks-contents
func (p *CssParser) ParseStyleBlockContents() any {
	return nil
}

// For the contents of a style attribute, which parses text into the contents of a single style rule.

// Despite the name, this actually parses a mixed list of declarations and at-rules, as CSS 2.1 does for @page. Unexpected at-rules (which could be all of them, in a given context) are invalid and will be ignored by the consumer.
//
// This algorithm does not handle nested style rules. If your use requires that, use parse a style block’s contents.
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-list-of-declarations
func (p *CssParser) ParseListOfDeclarations() []any {
	return nil
}

// For things that need to consume a single value, like the parsing rules for attr().
//
// @see https://www.w3.org/TR/css-syntax-3/#parse-component-value
func (p *CssParser) ParseComponentValue() (any, error) {
	return nil, nil
}

// for the contents of presentational attributes, which parse text into a single declaration’s value,
// or for parsing a stand-alone selector [SELECT](https://www.w3.org/TR/css-syntax-3/#biblio-select) or list of Media Queries [MEDIAQ](https://www.w3.org/TR/css-syntax-3/#biblio-mediaq),
// as in Selectors API or the media HTML attribute.
//
// https://www.w3.org/TR/css-syntax-3/#parse-list-of-component-values
func (p *CssParser) ParseListOfComponentValues() []any {
	return nil
}

//#endregion

// #region algorithms
// https://www.w3.org/TR/css-syntax-3/#consume-list-of-rules
func (p *CssParser) consumeListOfRules() {}

// https://www.w3.org/TR/css-syntax-3/#consume-at-rule
func (p *CssParser) consumeAtRule() {}

// https://www.w3.org/TR/css-syntax-3/#consume-qualified-rule
func (p *CssParser) consumeQualifiedRule() {}

// https://www.w3.org/TR/css-syntax-3/#consume-style-block
func (p *CssParser) consumeStyleBlockContents() {}

// https://www.w3.org/TR/css-syntax-3/#consume-list-of-declarations
func (p *CssParser) consumeListOfDeclarations() {}

// https://www.w3.org/TR/css-syntax-3/#consume-declaration
func (p *CssParser) consumeDeclaration() {}

// https://www.w3.org/TR/css-syntax-3/#consume-component-value
func (p *CssParser) consumeComponentValue() {}

// https://www.w3.org/TR/css-syntax-3/#consume-simple-block
func (p *CssParser) consumeSimpleBlock() {}

// https://www.w3.org/TR/css-syntax-3/#consume-function
func (p *CssParser) consumeFunction() {}

//#endregion
