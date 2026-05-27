package layout

import (
	"strings"

	"github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/css/tokenizer"
	"github.com/VisualSource/plex/internal/dom"
)

type Selector struct {
	List        SelectorList
	Specificity int // packed max specificity over List
}

// parseSelectorGrammar matches a list of component values against the
// <selector-list> grammar, returning the parsed list or parser.ErrSyntax. It is
// the matchGrammar callback handed to parser.ParseAccordingToCssGrammar.
//
// @see https://drafts.csswg.org/selectors/#grammar
func parseSelectorGrammar(components []tokenizer.Token) (SelectorList, error) {
	return parseSelectorList(components)
}

// NewSelector parses source as a <selector-list> and, on success, returns a
// Selector carrying the parsed list and its (max) specificity.
//
// @see https://www.w3.org/TR/selectors-4/#parse-selector
func NewSelector(selector string) (*Selector, error) {
	list, err := parser.ParseAccordingToCssGrammar(selector, parseSelectorGrammar)
	if err != nil {
		return nil, err
	}

	s := &Selector{List: list}
	for _, complex := range list {
		if v := complex.Specificity().Value(); v > s.Specificity {
			s.Specificity = v
		}
	}
	return s, nil
}

func (s *Selector) Matches(node dom.ElementNode, scopingRoots ...any) bool {

	for _, selector := range s.List {
		for _, compx := range selector.Units {

			// test simple selectors
			// only one compund selector in complex selecotr -> true
			//

		}
	}

	return false
}

//#region grammar

// parseSelectorList is <complex-selector-list> = <complex-selector>#.
func parseSelectorList(components []tokenizer.Token) (SelectorList, error) {
	segments := splitTopLevelComma(components)

	out := make(SelectorList, 0, len(segments))
	for _, seg := range segments {
		cs, err := parseComplexSelector(seg)
		if err != nil {
			return nil, err
		}
		out = append(out, cs)
	}
	return out, nil
}

// splitTopLevelComma splits a component-value list on bare comma tokens. This is
// safe at the top level because commas inside [...] / (...) are already grouped
// into SimpleBlock / Function values by consumeComponentValue. An empty input
// yields a single empty segment (which parseComplexSelector rejects).
func splitTopLevelComma(toks []tokenizer.Token) [][]tokenizer.Token {
	groups := make([][]tokenizer.Token, 0, 1)
	cur := make([]tokenizer.Token, 0)
	for _, t := range toks {
		if t.IsToken() == tokenizer.TokenId_Comma {
			groups = append(groups, cur)
			cur = make([]tokenizer.Token, 0)
			continue
		}
		cur = append(cur, t)
	}
	return append(groups, cur)
}

// parseComplexSelector is <complex-selector-unit> [ <combinator>? <complex-selector-unit> ]*.
//
// Whitespace separating two units with no explicit combinator is the descendant
// combinator; whitespace adjacent to an explicit combinator is insignificant.
func parseComplexSelector(seg []tokenizer.Token) (*ComplexSelector, error) {
	r := newTokenReader(seg)
	r.skipWhitespace()
	if r.atEnd() {
		return nil, parser.ErrSyntax
	}

	cs := &ComplexSelector{}

	comp, err := parseCompoundSelector(r)
	if err != nil {
		return nil, err
	}
	cs.Units = append(cs.Units, ComplexSelectorUnit{Combinator: CombinatorNone, Compound: comp})

	for {
		hadWS := r.peekIsWhitespace()
		r.skipWhitespace()
		if r.atEnd() {
			break // trailing whitespace, end of selector
		}

		comb, ok := tryParseCombinator(r)
		switch {
		case ok:
			// explicit combinator; surrounding whitespace is insignificant
		case hadWS:
			comb = CombinatorDescendant
		default:
			return nil, parser.ErrSyntax
		}

		r.skipWhitespace()
		comp, err := parseCompoundSelector(r)
		if err != nil {
			return nil, err
		}
		cs.Units = append(cs.Units, ComplexSelectorUnit{Combinator: comb, Compound: comp})
	}

	return cs, nil
}

// tryParseCombinator is <combinator> = '>' | '+' | '~' | [ '|' '|' ]. A single
// '|' is never claimed here: it belongs to a namespace prefix and is handled by
// the following compound selector.
func tryParseCombinator(r *tokenReader) (Combinator, bool) {
	t := r.peek()
	switch {
	case parser.IsDelim(t, '>'):
		r.next()
		return CombinatorChild, true
	case parser.IsDelim(t, '+'):
		r.next()
		return CombinatorNextSibling, true
	case parser.IsDelim(t, '~'):
		r.next()
		return CombinatorSubsequentSibling, true
	case parser.IsDelim(t, '|') && parser.IsDelim(r.peekN(1), '|'):
		r.next()
		r.next()
		return CombinatorColumn, true
	default:
		return CombinatorNone, false
	}
}

// parseCompoundSelector is
//
//	[ <type-selector>? <subclass-selector>* [ <pseudo-element-selector> <pseudo-class-selector>* ]* ]!
//
// The trailing '!' requires the compound to match at least one simple selector.
func parseCompoundSelector(r *tokenReader) (*CompoundSelector, error) {
	c := &CompoundSelector{}
	matched := false

	if ts, ok, err := tryParseTypeSelector(r); err != nil {
		return nil, err
	} else if ok {
		c.Type = ts
		matched = true
	}

	for {
		ss, ok, err := tryParseSubclassSelector(r)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		c.Subclass = append(c.Subclass, ss)
		matched = true
	}

	for {
		pe, ok, err := tryParsePseudoElement(r)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		c.Pseudos = append(c.Pseudos, pe)
		matched = true

		// pseudo-classes may trail a pseudo-element (e.g. ::selection:hover)
		for {
			pc, ok, err := tryParsePseudoClass(r)
			if err != nil {
				return nil, err
			}
			if !ok {
				break
			}
			pe.Pseudos = append(pe.Pseudos, pc.(*PseudoClassSelector))
		}
	}

	if !matched {
		return nil, parser.ErrSyntax
	}
	return c, nil
}

// tryParseTypeSelector is <type-selector> = <wq-name> | <ns-prefix>? '*'.
func tryParseTypeSelector(r *tokenReader) (*TypeSelector, bool, error) {
	start := r.mark()
	ts := &TypeSelector{}

	if prefix, ok := tryParseNsPrefix(r); ok {
		ts.HasNamespace = true
		ts.Prefix = prefix
		// tryParseNsPrefix only succeeds when a name (ident or '*') follows.
		if name, ok := identValue(r.peek()); ok {
			r.next()
			ts.LocalName = name
			return ts, true, nil
		}
		if parser.IsDelim(r.peek(), '*') {
			r.next()
			ts.Universal = true
			return ts, true, nil
		}
		return nil, false, parser.ErrSyntax
	}

	if name, ok := identValue(r.peek()); ok {
		r.next()
		ts.LocalName = name
		return ts, true, nil
	}
	if parser.IsDelim(r.peek(), '*') {
		r.next()
		ts.Universal = true
		return ts, true, nil
	}

	r.reset(start)
	return nil, false, nil
}

// tryParseNsPrefix consumes <ns-prefix> = [ <ident-token> | '*' ]? '|', but only
// when a name (ident or '*') follows the '|'. This both keeps the '||' column
// combinator out and lets an attribute matcher like '|=' fall through (e.g. the
// '|' in "[a|=b]" is dash-match, not a namespace).
func tryParseNsPrefix(r *tokenReader) (string, bool) {
	start := r.mark()

	if name, ok := identValue(r.peek()); ok {
		if parser.IsDelim(r.peekN(1), '|') && nsNameFollows(r.peekN(2)) {
			r.next() // ident
			r.next() // '|'
			return name, true
		}
	} else if parser.IsDelim(r.peek(), '*') {
		if parser.IsDelim(r.peekN(1), '|') && nsNameFollows(r.peekN(2)) {
			r.next() // '*'
			r.next() // '|'
			return "*", true
		}
	} else if parser.IsDelim(r.peek(), '|') && nsNameFollows(r.peekN(1)) {
		r.next() // '|'
		return "", true
	}

	r.reset(start)
	return "", false
}

func nsNameFollows(t tokenizer.Token) bool {
	if t == nil {
		return false
	}
	return t.IsToken() == tokenizer.TokenId_Ident || parser.IsDelim(t, '*')
}

// tryParseSubclassSelector is
// <subclass-selector> = <id-selector> | <class-selector> | <attribute-selector> | <pseudo-class-selector>.
func tryParseSubclassSelector(r *tokenReader) (SimpleSelector, bool, error) {
	t := r.peek()
	if t == nil {
		return nil, false, nil
	}

	switch {
	case t.IsToken() == tokenizer.TokenId_Hash:
		h, ok := t.(*tokenizer.MultiCharacterToken)
		if !ok || h.Flag != "id" { // only id-typed hashes are id selectors
			return nil, false, parser.ErrSyntax
		}
		r.next()
		return &IdSelector{Name: h.Value}, true, nil

	case parser.IsDelim(t, '.'):
		r.next()
		name, ok := identValue(r.peek())
		if !ok {
			return nil, false, parser.ErrSyntax
		}
		r.next()
		return &ClassSelector{Name: name}, true, nil

	case t.IsToken() == tokenizer.TokenId_Colon:
		// '::' introduces a pseudo-element, handled by parseCompoundSelector.
		if next := r.peekN(1); next != nil && next.IsToken() == tokenizer.TokenId_Colon {
			return nil, false, nil
		}
		return tryParsePseudoClass(r)
	}

	if b, ok := t.(*parser.SimpleBlock); ok && b.StartDelim != nil &&
		b.StartDelim.IsToken() == tokenizer.TokenId_BracketSquareOpen {
		r.next()
		attr, err := parseAttributeSelector(b)
		if err != nil {
			return nil, false, err
		}
		return attr, true, nil
	}

	return nil, false, nil
}

// parseAttributeSelector parses the contents of a '[...]' SimpleBlock as
//
//	'[' <wq-name> ']' | '[' <wq-name> <attr-matcher> [ <string> | <ident> ] <attr-modifier>? ']'
func parseAttributeSelector(block *parser.SimpleBlock) (*AttributeSelector, error) {
	r := newTokenReader(block.Value)
	r.skipWhitespace()

	attr := &AttributeSelector{}

	if prefix, ok := tryParseNsPrefix(r); ok {
		attr.HasNamespace = true
		attr.Prefix = prefix
	}

	name, ok := identValue(r.peek())
	if !ok {
		return nil, parser.ErrSyntax
	}
	r.next()
	attr.LocalName = name
	r.skipWhitespace()

	// '[' <wq-name> ']' — presence only
	if r.atEnd() {
		attr.Matcher = AttrPresence
		return attr, nil
	}

	matcher, err := parseAttrMatcher(r)
	if err != nil {
		return nil, err
	}
	attr.Matcher = matcher
	r.skipWhitespace()

	// value: <string-token> | <ident-token>
	v, ok := r.peek().(*tokenizer.MultiCharacterToken)
	if !ok || (v.Type != tokenizer.TokenId_String && v.Type != tokenizer.TokenId_Ident) {
		return nil, parser.ErrSyntax
	}
	attr.Value = v.Value
	r.next()
	r.skipWhitespace()

	// optional case-sensitivity modifier: i | s
	if mod, ok := identValue(r.peek()); ok {
		switch {
		case strings.EqualFold(mod, "i"):
			attr.Modifier = AttrModCaseInsensitive
		case strings.EqualFold(mod, "s"):
			attr.Modifier = AttrModCaseSensitive
		default:
			return nil, parser.ErrSyntax
		}
		r.next()
		r.skipWhitespace()
	}

	if !r.atEnd() {
		return nil, parser.ErrSyntax
	}
	return attr, nil
}

// parseAttrMatcher is <attr-matcher> = [ '~' | '|' | '^' | '$' | '*' ]? '='.
func parseAttrMatcher(r *tokenReader) (AttrMatcher, error) {
	matcher := AttrEquals
	switch {
	case parser.IsDelim(r.peek(), '~'):
		matcher = AttrIncludes
		r.next()
	case parser.IsDelim(r.peek(), '|'):
		matcher = AttrDashMatch
		r.next()
	case parser.IsDelim(r.peek(), '^'):
		matcher = AttrPrefix
		r.next()
	case parser.IsDelim(r.peek(), '$'):
		matcher = AttrSuffix
		r.next()
	case parser.IsDelim(r.peek(), '*'):
		matcher = AttrSubstring
		r.next()
	}
	if !parser.IsDelim(r.peek(), '=') {
		return 0, parser.ErrSyntax
	}
	r.next()
	return matcher, nil
}

// tryParsePseudoElement handles '::ident' and the four legacy single-colon
// pseudo-elements (:before, :after, :first-line, :first-letter).
func tryParsePseudoElement(r *tokenReader) (*PseudoElementSelector, bool, error) {
	t := r.peek()
	if t == nil || t.IsToken() != tokenizer.TokenId_Colon {
		return nil, false, nil
	}

	// '::' modern pseudo-element
	if next := r.peekN(1); next != nil && next.IsToken() == tokenizer.TokenId_Colon {
		r.next() // ':'
		r.next() // ':'
		name, ok := identValue(r.peek())
		if !ok {
			return nil, false, parser.ErrSyntax
		}
		r.next()
		return &PseudoElementSelector{Name: strings.ToLower(name)}, true, nil
	}

	// single ':' legacy pseudo-element
	if name, ok := identValue(r.peekN(1)); ok && isLegacyPseudoElement(name) {
		r.next() // ':'
		r.next() // ident
		return &PseudoElementSelector{Name: strings.ToLower(name), Legacy: true}, true, nil
	}

	return nil, false, nil
}

// tryParsePseudoClass is ':' <ident-token> | ':' <function-token> <any-value> ')'.
// Functional pseudo-classes keep their raw argument tokens (no recursion, no
// An+B parsing in phase 1).
func tryParsePseudoClass(r *tokenReader) (SimpleSelector, bool, error) {
	t := r.peek()
	if t == nil || t.IsToken() != tokenizer.TokenId_Colon {
		return nil, false, nil
	}
	// never claim '::' (pseudo-element)
	if next := r.peekN(1); next != nil && next.IsToken() == tokenizer.TokenId_Colon {
		return nil, false, nil
	}

	next := r.peekN(1)

	if name, ok := identValue(next); ok {
		// a single-colon legacy pseudo-element name is not a pseudo-class
		if isLegacyPseudoElement(name) {
			return nil, false, nil
		}
		r.next() // ':'
		r.next() // ident
		return &PseudoClassSelector{Name: strings.ToLower(name)}, true, nil
	}

	if fn, ok := next.(*parser.Function); ok {
		r.next() // ':'
		r.next() // function
		return &PseudoClassSelector{
			Name:       strings.ToLower(fn.Name),
			Functional: true,
			RawArgs:    fn.Value,
		}, true, nil
	}

	return nil, false, parser.ErrSyntax
}

func isLegacyPseudoElement(name string) bool {
	switch strings.ToLower(name) {
	case "before", "after", "first-line", "first-letter":
		return true
	}
	return false
}

// identValue returns the value of an <ident-token>, or ("", false) for any other
// token (including nil). It composes parser.GetTokenValueAsString rather than
// reimplementing it.
func identValue(t tokenizer.Token) (string, bool) {
	if t != nil && t.IsToken() == tokenizer.TokenId_Ident {
		return parser.GetTokenValueAsString(t), true
	}
	return "", false
}

//#endregion
