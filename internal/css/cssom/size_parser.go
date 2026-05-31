package cssom

import (
	"errors"
	"strings"

	css_parser "github.com/VisualSource/plex/internal/css/parser"
	css_tokenizer "github.com/VisualSource/plex/internal/css/tokenizer"
)

// LengthPercentage holds a CSS length or percentage value.
type LengthPercentage struct {
	Value float64
	Unit  string // "%" for percentage; CSS length unit (px, em, rem…) otherwise
}

// CalcNode is a node in a CSS calc expression tree.
type CalcNode interface{ isCalcNode() }

// CalcLeaf is a terminal value in a calc expression.
// Kind is one of: "number", "dimension", "percentage",
// "e", "pi", "infinity", "-infinity", "NaN".
type CalcLeaf struct {
	Kind  string
	Value float64
	Unit  string
}

func (CalcLeaf) isCalcNode() {}

// CalcBinary is a binary operation node (+, -, *, /).
type CalcBinary struct {
	Op          rune
	Left, Right CalcNode
}

func (CalcBinary) isCalcNode() {}

// CalcSizeBasis is the first argument of calc-size().
// Kind is one of: "any", "size-keyword", "calc-size", "calc-sum".
type CalcSizeBasis struct {
	Kind    string
	Keyword string         // for "size-keyword"
	Nested  *CalcSizeValue // for "calc-size"
	Expr    CalcNode       // for "calc-sum"
}

// CalcSizeValue represents a parsed calc-size() expression.
type CalcSizeValue struct {
	Basis CalcSizeBasis
	Expr  CalcNode
}

// AnchorSizeValue represents a parsed anchor-size() expression.
type AnchorSizeValue struct {
	Name     string            // dashed-ident anchor name, may be ""
	Size     string            // width|height|block|inline|self-block|self-inline, may be ""
	Fallback *LengthPercentage // optional fallback, may be nil
}

// SizeKind identifies which variant of Size is active.
type SizeKind uint

const (
	SizeKindKeyword      SizeKind = iota // auto|min-content|max-content|fit-content|stretch|contain
	SizeKindLP                           // <length-percentage>
	SizeKindFitContentFn                 // fit-content(<length-percentage>)
	SizeKindCalcSize                     // calc-size(…)
	SizeKindAnchorSize                   // anchor-size(…)
)

// Size represents a parsed CSS width/height value.
type Size struct {
	Kind       SizeKind
	Keyword    string
	LP         *LengthPercentage
	CalcSize   *CalcSizeValue
	AnchorSize *AnchorSizeValue
}

// splitComma splits tokens on top-level Comma tokens, returning
// each part with leading/trailing whitespace stripped.
func splitComma(tokens []css_tokenizer.Token) [][]css_tokenizer.Token {
	var parts [][]css_tokenizer.Token
	var cur []css_tokenizer.Token
	for _, t := range tokens {
		if t.IsToken() == css_tokenizer.TokenId_Comma {
			parts = append(parts, nonWS(cur))
			cur = nil
		} else {
			cur = append(cur, t)
		}
	}
	parts = append(parts, nonWS(cur))
	return parts
}

func parseLengthPercentage(tokens []css_tokenizer.Token) (*LengthPercentage, error) {
	t := nonWS(tokens)
	if len(t) != 1 {
		return nil, errors.New("expected single length or percentage")
	}
	switch t[0].IsToken() {
	case css_tokenizer.TokenId_Dimension:
		nt := t[0].(*css_tokenizer.NumericToken)
		if nt.Value < 0 {
			return nil, errors.New("negative length not allowed")
		}
		return &LengthPercentage{Value: nt.Value, Unit: nt.Unit}, nil
	case css_tokenizer.TokenId_Percentage:
		nt := t[0].(*css_tokenizer.NumericToken)
		if nt.Value < 0 {
			return nil, errors.New("negative percentage not allowed")
		}
		return &LengthPercentage{Value: nt.Value, Unit: "%"}, nil
	case css_tokenizer.TokenId_Number:
		nt := t[0].(*css_tokenizer.NumericToken)
		if nt.Value == 0 {
			return &LengthPercentage{Value: 0}, nil
		}
	}
	return nil, errors.New("expected length or percentage")
}

var calcKeywords = map[string]bool{
	"e": true, "pi": true, "infinity": true, "-infinity": true, "nan": true,
}

func parseCalcValue(s *tokenScanner) (CalcNode, error) {
	s.skipWS()
	t := s.peek()
	if t == nil {
		return nil, errors.New("unexpected end in calc expression")
	}
	switch t.IsToken() {
	case css_tokenizer.TokenId_Number:
		s.next()
		nt := t.(*css_tokenizer.NumericToken)
		return CalcLeaf{Kind: "number", Value: nt.Value}, nil
	case css_tokenizer.TokenId_Dimension:
		s.next()
		nt := t.(*css_tokenizer.NumericToken)
		return CalcLeaf{Kind: "dimension", Value: nt.Value, Unit: nt.Unit}, nil
	case css_tokenizer.TokenId_Percentage:
		s.next()
		nt := t.(*css_tokenizer.NumericToken)
		return CalcLeaf{Kind: "percentage", Value: nt.Value}, nil
	case css_tokenizer.TokenId_Ident:
		name := strings.ToLower(css_parser.GetTokenValueAsString(t))
		if calcKeywords[name] {
			s.next()
			return CalcLeaf{Kind: name}, nil
		}
	case css_parser.TokenId_SimpleBlock:
		blk := t.(*css_parser.SimpleBlock)
		if blk.StartDelim.IsToken() == css_tokenizer.TokenId_BracketParamOpen {
			s.next()
			inner := &tokenScanner{tokens: blk.Value}
			node, err := parseCalcSum(inner)
			if err != nil {
				return nil, err
			}
			return node, nil
		}
	}
	return nil, errors.New("invalid calc value")
}

func parseCalcProduct(s *tokenScanner) (CalcNode, error) {
	left, err := parseCalcValue(s)
	if err != nil {
		return nil, err
	}
	for {
		s.skipWS()
		t := s.peek()
		if t == nil || t.IsToken() != css_tokenizer.TokenId_Delim {
			break
		}
		delim, ok := t.(*css_tokenizer.SingleCharacterToken)
		if !ok || (delim.Value != '*' && delim.Value != '/') {
			break
		}
		s.next()
		right, err := parseCalcValue(s)
		if err != nil {
			return nil, err
		}
		left = CalcBinary{Op: delim.Value, Left: left, Right: right}
	}
	return left, nil
}

func parseCalcSum(s *tokenScanner) (CalcNode, error) {
	left, err := parseCalcProduct(s)
	if err != nil {
		return nil, err
	}
	for {
		s.skipWS()
		t := s.peek()
		if t == nil || t.IsToken() != css_tokenizer.TokenId_Delim {
			break
		}
		delim, ok := t.(*css_tokenizer.SingleCharacterToken)
		if !ok || (delim.Value != '+' && delim.Value != '-') {
			break
		}
		s.next()
		right, err := parseCalcProduct(s)
		if err != nil {
			return nil, err
		}
		left = CalcBinary{Op: delim.Value, Left: left, Right: right}
	}
	return left, nil
}

var sizeKeywords = map[string]bool{
	"min-content": true, "max-content": true, "fit-content": true,
	"stretch": true, "contain": true,
}

func parseCalcSizeFn(innerTokens []css_tokenizer.Token) (*CalcSizeValue, error) {
	parts := splitComma(innerTokens)
	if len(parts) != 2 {
		return nil, errors.New("calc-size() requires exactly two arguments")
	}

	// parse basis (first argument)
	basisTokens := parts[0]
	var basis CalcSizeBasis
	if len(basisTokens) == 1 {
		tok := basisTokens[0]
		switch tok.IsToken() {
		case css_tokenizer.TokenId_Ident:
			name := strings.ToLower(css_parser.GetTokenValueAsString(tok))
			if name == "any" {
				basis = CalcSizeBasis{Kind: "any"}
			} else if sizeKeywords[name] {
				basis = CalcSizeBasis{Kind: "size-keyword", Keyword: name}
			}
		case css_parser.TokenId_Function:
			fn := tok.(*css_parser.Function)
			if strings.EqualFold(fn.Name, "calc-size") {
				nested, err := parseCalcSizeFn(fn.Value)
				if err != nil {
					return nil, err
				}
				basis = CalcSizeBasis{Kind: "calc-size", Nested: nested}
			}
		}
	}
	if basis.Kind == "" {
		s := &tokenScanner{tokens: basisTokens}
		expr, err := parseCalcSum(s)
		if err != nil {
			return nil, errors.New("invalid calc-size basis")
		}
		basis = CalcSizeBasis{Kind: "calc-sum", Expr: expr}
	}

	// parse expression (second argument)
	s := &tokenScanner{tokens: parts[1]}
	expr, err := parseCalcSum(s)
	if err != nil {
		return nil, err
	}

	return &CalcSizeValue{Basis: basis, Expr: expr}, nil
}

var anchorSizeKeywords = map[string]bool{
	"width": true, "height": true, "block": true,
	"inline": true, "self-block": true, "self-inline": true,
}

// <anchor-size()> =
// anchor-size( [ <anchor-name> || <anchor-size> ]? , <length-percentage>? )
func parseAnchorSizeFn(innerTokens []css_tokenizer.Token) (*AnchorSizeValue, error) {
	parts := splitComma(innerTokens)
	if len(parts) > 2 {
		return nil, errors.New("anchor-size() takes at most two comma-separated arguments")
	}

	result := &AnchorSizeValue{}

	// first part: [ <anchor-name> || <anchor-size> ]?
	if len(parts) >= 1 {
		for _, tok := range parts[0] {
			if tok.IsToken() != css_tokenizer.TokenId_Ident {
				continue
			}
			val := css_parser.GetTokenValueAsString(tok)
			lower := strings.ToLower(val)
			switch {
			case strings.HasPrefix(val, "--"):
				result.Name = val
			case anchorSizeKeywords[lower]:
				result.Size = lower
			}
		}
	}

	// second part: <length-percentage>?
	if len(parts) == 2 && len(parts[1]) > 0 {
		lp, err := parseLengthPercentage(parts[1])
		if err != nil {
			return nil, err
		}
		result.Fallback = lp
	}

	return result, nil
}

/*
height,width =

	auto                                      |
	<length-percentage [0,∞]>                 |
	min-content                               |
	max-content                               |
	fit-content( <length-percentage [0,∞]> )  |
	<calc-size()>                             |
	<anchor-size()>                           |
	stretch                                   |
	fit-content                               |
	contain

<length-percentage> =

	<length>      |
	<percentage>

<calc-size()> =

	calc-size( <calc-size-basis> , <calc-sum> )

<anchor-size()> =

	anchor-size( [ <anchor-name> || <anchor-size> ]? , <length-percentage>? )

<calc-size-basis> =

	<size-keyword>  |
	<calc-size()>   |
	any             |
	<calc-sum>

<calc-sum> =

	<calc-product> [ [ '+' | '-' ] <calc-product> ]*

<anchor-name> =

	<dashed-ident>

<anchor-size> =

	width        |
	height       |
	block        |
	inline       |
	self-block   |
	self-inline

<calc-product> =

	<calc-value> [ [ '*' | / ] <calc-value> ]*

<calc-value> =

	<number>        |
	<dimension>     |
	<percentage>    |
	<calc-keyword>  |
	( <calc-sum> )

<calc-keyword> =

	e          |
	pi         |
	infinity   |
	-infinity  |
	NaN
*/
func parseSize(tokens []css_tokenizer.Token) (Size, error) {
	t := nonWS(tokens)
	if len(t) == 0 {
		return Size{}, ErrInvalidCssValue
	}
	if len(t) != 1 {
		return Size{}, ErrInvalidCssValue
	}

	tok := t[0]
	switch tok.IsToken() {
	case css_tokenizer.TokenId_Ident:
		name := strings.ToLower(css_parser.GetTokenValueAsString(tok))
		switch name {
		case "auto", "min-content", "max-content", "fit-content", "stretch", "contain":
			return Size{Kind: SizeKindKeyword, Keyword: name}, nil
		}

	case css_tokenizer.TokenId_Dimension, css_tokenizer.TokenId_Percentage:
		lp, err := parseLengthPercentage(t)
		if err != nil {
			return Size{}, err
		}
		return Size{Kind: SizeKindLP, LP: lp}, nil

	case css_tokenizer.TokenId_Number:
		nt := tok.(*css_tokenizer.NumericToken)
		if nt.Value == 0 {
			return Size{Kind: SizeKindLP, LP: &LengthPercentage{Value: 0}}, nil
		}

	case css_parser.TokenId_Function:
		fn := tok.(*css_parser.Function)
		switch strings.ToLower(fn.Name) {
		case "fit-content":
			lp, err := parseLengthPercentage(fn.Value)
			if err != nil {
				return Size{}, err
			}
			return Size{Kind: SizeKindFitContentFn, LP: lp}, nil
		case "calc-size":
			cs, err := parseCalcSizeFn(fn.Value)
			if err != nil {
				return Size{}, err
			}
			return Size{Kind: SizeKindCalcSize, CalcSize: cs}, nil
		case "anchor-size":
			as, err := parseAnchorSizeFn(fn.Value)
			if err != nil {
				return Size{}, err
			}
			return Size{Kind: SizeKindAnchorSize, AnchorSize: as}, nil
		}
	}

	return Size{}, ErrInvalidCssValue
}
