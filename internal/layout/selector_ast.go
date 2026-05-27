package layout

import "github.com/VisualSource/plex/internal/css/tokenizer"

// SelectorList is a <complex-selector-list> = <complex-selector>#
//
// @see https://drafts.csswg.org/selectors/#typedef-selector-list
type SelectorList []*ComplexSelector

// ComplexSelector is <complex-selector-unit> [ <combinator>? <complex-selector-unit> ]*
//
// Units are stored left-to-right as written. Units[0] always carries
// CombinatorNone; every later unit carries the combinator connecting it to the
// preceding unit.
type ComplexSelector struct {
	Units []ComplexSelectorUnit
}

type Combinator int

const (
	CombinatorNone              Combinator = iota // only valid on Units[0]
	CombinatorDescendant                          // ' ' (whitespace)
	CombinatorChild                               // '>'
	CombinatorNextSibling                         // '+'
	CombinatorSubsequentSibling                   // '~'
	CombinatorColumn                              // '||'
)

// ComplexSelectorUnit is a single compound selector together with the
// combinator that connects it to the previous unit in the chain.
type ComplexSelectorUnit struct {
	Combinator Combinator
	Compound   *CompoundSelector
}

// CompoundSelector is
//
//	[ <type-selector>? <subclass-selector>* [ <pseudo-element-selector> <pseudo-class-selector>* ]* ]!
//
// A nil Type means no explicit type selector was written (implicitly universal
// at match time).
type CompoundSelector struct {
	Type     *TypeSelector    // nil => implicitly universal
	Subclass []SimpleSelector // id / class / attribute / pseudo-class, in source order
	Pseudos  []SimpleSelector // pseudo-element(s) and their trailing pseudo-classes
}

// SimpleSelector is the sum type of all simple selectors that can appear inside
// a compound selector.
type SimpleSelector interface{ isSimpleSelector() }

// TypeSelector is <wq-name> | <ns-prefix>? '*'.
//
// HasNamespace reports whether a '|' was present:
//   - HasNamespace=false           -> no prefix written ("div")
//   - HasNamespace=true, Prefix=""  -> no-namespace ("|div")
//   - Prefix="*"                    -> any-namespace ("*|div")
//   - Prefix="foo"                  -> literal prefix ("foo|div")
type TypeSelector struct {
	HasNamespace bool
	Prefix       string
	Universal    bool   // '*' (LocalName is ignored)
	LocalName    string // tag name when !Universal
}

func (*TypeSelector) isSimpleSelector() {}

// IdSelector is <id-selector> = <hash-token> (only hashes with the "id" type flag).
type IdSelector struct{ Name string }

func (*IdSelector) isSimpleSelector() {}

// ClassSelector is <class-selector> = '.' <ident-token>.
type ClassSelector struct{ Name string }

func (*ClassSelector) isSimpleSelector() {}

type AttrMatcher int

const (
	AttrPresence  AttrMatcher = iota // [attr]
	AttrEquals                       // =
	AttrIncludes                     // ~=
	AttrDashMatch                    // |=
	AttrPrefix                       // ^=
	AttrSuffix                       // $=
	AttrSubstring                    // *=
)

type AttrModifier int

const (
	AttrModNone           AttrModifier = iota
	AttrModCaseInsensitive              // i
	AttrModCaseSensitive                // s
)

// AttributeSelector is <attribute-selector>.
type AttributeSelector struct {
	HasNamespace bool
	Prefix       string
	LocalName    string
	Matcher      AttrMatcher
	Value        string // empty when Matcher == AttrPresence
	Modifier     AttrModifier
}

func (*AttributeSelector) isSimpleSelector() {}

// PseudoClassSelector is ':' <ident-token> | ':' <function-token> <any-value> ')'.
//
// For functional pseudo-classes the raw inner component values are kept in
// RawArgs; phase-1 performs no recursion into argument selector lists and no
// An+B parsing.
type PseudoClassSelector struct {
	Name       string            // lower-cased, e.g. "hover", "not", "nth-child"
	Functional bool              // true if it was ':name(...)'
	RawArgs    []tokenizer.Token // raw inner component values when Functional
}

func (*PseudoClassSelector) isSimpleSelector() {}

// PseudoElementSelector is <pseudo-element-selector>, including the legacy
// single-colon forms (:before, :after, :first-line, :first-letter).
type PseudoElementSelector struct {
	Name    string                 // lower-cased
	Legacy  bool                   // matched via the single-colon legacy form
	Pseudos []*PseudoClassSelector // trailing user-action pseudo-classes
}

func (*PseudoElementSelector) isSimpleSelector() {}

// Specificity is the (A, B, C) triple defined by
// https://drafts.csswg.org/selectors/#specificity-rules.
type Specificity struct{ A, B, C int }

// Value packs the triple into a single comparable integer using the
// conventional weighting used by browser engines.
func (s Specificity) Value() int { return s.A*0x10000 + s.B*0x100 + s.C }

// Specificity computes the specificity of a complex selector by walking its
// compound selectors: A counts id selectors, B counts class, attribute and
// pseudo-class selectors, and C counts (non-universal) type selectors and
// pseudo-elements.
//
// Note: the special specificity rules for :is/:not/:where/:has and
// :nth-child(... of S) are phase-2 work; every pseudo-class currently counts
// flat toward B.
func (c *ComplexSelector) Specificity() Specificity {
	var s Specificity
	for _, unit := range c.Units {
		comp := unit.Compound
		if comp == nil {
			continue
		}
		if comp.Type != nil && !comp.Type.Universal {
			s.C++
		}
		for _, sub := range comp.Subclass {
			switch sub.(type) {
			case *IdSelector:
				s.A++
			case *ClassSelector, *AttributeSelector, *PseudoClassSelector:
				s.B++
			}
		}
		for _, pseudo := range comp.Pseudos {
			switch p := pseudo.(type) {
			case *PseudoElementSelector:
				s.C++
				s.B += len(p.Pseudos)
			case *PseudoClassSelector:
				s.B++
			}
		}
	}
	return s
}
