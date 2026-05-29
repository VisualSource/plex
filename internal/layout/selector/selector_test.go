package selector_test

import (
	"testing"

	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/layout/selector"
	"github.com/VisualSource/plex/internal/utils"
)

func TestNewSelector_AcceptReject(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		// accept
		{"div", false},
		{".foo", false},
		{"#bar", false},
		{"a.b#c", false},
		{"ul > li", false},
		{"a + b ~ c", false},
		{"a||b", false},
		{"h1, h2, h3", false},
		{"a , b", false},
		{"[href]", false},
		{`[href="x"]`, false},
		{`[href^="https" i]`, false},
		{`a[rel~="next"]`, false},
		{"[a|=b]", false},
		{":hover", false},
		{"a:hover::before", false},
		{"::before", false},
		{":before", false},
		{":not(.x)", false},
		{":nth-child(2n+1)", false},
		{"*", false},
		{"*|*", false},
		{"|div", false},

		// reject
		{"", true},
		{",div", true},
		{"div,", true},
		{"div >", true},
		{"> div", true},
		{".", true},
		{"a:", true},
		{"div ||", true},
		{"[]", true},
		{`[href=]`, true},
		{"#123", true},
		{"a!b", true},
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			_, err := selector.NewSelector(c.in)
			switch {
			case c.wantErr && err != nil:
				t.Fatalf("NewSelector(%q): want ErrSyntax, got %v", c.in, err)
			case !c.wantErr && err != nil:
				t.Fatalf("NewSelector(%q): unexpected error: %v", c.in, err)
			}
		})
	}
}

func TestNewSelector_Structure(t *testing.T) {
	sel, err := selector.NewSelector("ul.menu > li#x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sel.List) != 1 {
		t.Fatalf("complex selectors = %d, want 1", len(sel.List))
	}

	cs := sel.List[0]
	if len(cs.Units) != 2 {
		t.Fatalf("units = %d, want 2", len(cs.Units))
	}
	if cs.Units[0].Combinator != selector.CombinatorNone {
		t.Fatalf("unit[0] combinator = %v, want None", cs.Units[0].Combinator)
	}
	if cs.Units[1].Combinator != selector.CombinatorChild {
		t.Fatalf("unit[1] combinator = %v, want Child", cs.Units[1].Combinator)
	}
	if got := cs.Units[0].Compound.Type; got == nil || got.LocalName != "ul" {
		t.Fatalf("unit[0] type = %+v, want ul", got)
	}
	if got := cs.Units[1].Compound.Type; got == nil || got.LocalName != "li" {
		t.Fatalf("unit[1] type = %+v, want li", got)
	}

	// ul.menu => c=1,b=1 ; li#x => c=1,a=1 ; total a=1,b=1,c=2
	want := selector.Specificity{A: 1, B: 1, C: 2}
	if got := cs.Specificity(); got != want {
		t.Fatalf("specificity = %+v, want %+v", got, want)
	}
	if sel.Specificity != want.Value() {
		t.Fatalf("Selector.Specificity = %d, want %d", sel.Specificity, want.Value())
	}
}

func TestNewSelector_SpecificityMaxOverList(t *testing.T) {
	// "div, #id" -> max( c=1 , a=1 ) = a=1
	sel, err := selector.NewSelector("div, #id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := (selector.Specificity{A: 1}).Value(); sel.Specificity != want {
		t.Fatalf("Selector.Specificity = %d, want %d", sel.Specificity, want)
	}
}

func firstCompound(t *testing.T, in string) *selector.CompoundSelector {
	t.Helper()
	sel, err := selector.NewSelector(in)
	if err != nil {
		t.Fatalf("NewSelector(%q): unexpected error: %v", in, err)
	}
	return sel.List[0].Units[0].Compound
}

func TestParseAttributeSelector(t *testing.T) {
	cases := []struct {
		in       string
		matcher  selector.AttrMatcher
		value    string
		modifier selector.AttrModifier
	}{
		{"[href]", selector.AttrPresence, "", selector.AttrModNone},
		{`[href="x"]`, selector.AttrEquals, "x", selector.AttrModNone},
		{`[rel~="next"]`, selector.AttrIncludes, "next", selector.AttrModNone},
		{"[lang|=en]", selector.AttrDashMatch, "en", selector.AttrModNone},
		{`[href^="https"]`, selector.AttrPrefix, "https", selector.AttrModNone},
		{`[src$=".png"]`, selector.AttrSuffix, ".png", selector.AttrModNone},
		{`[title*="x"]`, selector.AttrSubstring, "x", selector.AttrModNone},
		{`[href="X" i]`, selector.AttrEquals, "X", selector.AttrModCaseInsensitive},
		{`[href="X" s]`, selector.AttrEquals, "X", selector.AttrModCaseSensitive},
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			comp := firstCompound(t, c.in)
			if len(comp.Subclass) != 1 {
				t.Fatalf("subclass count = %d, want 1", len(comp.Subclass))
			}
			attr, ok := comp.Subclass[0].(*selector.AttributeSelector)
			if !ok {
				t.Fatalf("subclass[0] = %T, want *AttributeSelector", comp.Subclass[0])
			}
			if attr.Matcher != c.matcher {
				t.Errorf("matcher = %v, want %v", attr.Matcher, c.matcher)
			}
			if attr.Value != c.value {
				t.Errorf("value = %q, want %q", attr.Value, c.value)
			}
			if attr.Modifier != c.modifier {
				t.Errorf("modifier = %v, want %v", attr.Modifier, c.modifier)
			}
		})
	}
}

func TestParsePseudo(t *testing.T) {
	// :hover -> pseudo-class in Subclass
	comp := firstCompound(t, "a:hover")
	if len(comp.Subclass) != 1 {
		t.Fatalf(":hover subclass count = %d, want 1", len(comp.Subclass))
	}
	if pc, ok := comp.Subclass[0].(*selector.PseudoClassSelector); !ok || pc.Name != "hover" || pc.Functional {
		t.Fatalf(":hover = %+v", comp.Subclass[0])
	}

	// :before -> legacy pseudo-element in Pseudos
	comp = firstCompound(t, ":before")
	if len(comp.Pseudos) != 1 {
		t.Fatalf(":before pseudos count = %d, want 1", len(comp.Pseudos))
	}
	if pe, ok := comp.Pseudos[0].(*selector.PseudoElementSelector); !ok || pe.Name != "before" || !pe.Legacy {
		t.Fatalf(":before = %+v", comp.Pseudos[0])
	}

	// ::before -> modern pseudo-element
	comp = firstCompound(t, "::before")
	if pe, ok := comp.Pseudos[0].(*selector.PseudoElementSelector); !ok || pe.Name != "before" || pe.Legacy {
		t.Fatalf("::before = %+v", comp.Pseudos[0])
	}

	// :not(.x) -> functional pseudo-class with raw args
	comp = firstCompound(t, ":not(.x)")
	pc, ok := comp.Subclass[0].(*selector.PseudoClassSelector)
	if !ok || pc.Name != "not" || !pc.Functional {
		t.Fatalf(":not(.x) = %+v", comp.Subclass[0])
	}
	if len(pc.RawArgs) == 0 {
		t.Fatalf(":not(.x) raw args are empty")
	}
}

// newEl builds a detached HTML element with the given tag and key/value
// attribute pairs (e.g. newEl(t, "div", "class", "a b", "id", "x")).
func newEl(t *testing.T, tag string, attrs ...string) dom.ElementNode {
	t.Helper()
	if len(attrs)%2 != 0 {
		t.Fatalf("newEl(%q): attrs must be key/value pairs, got %d", tag, len(attrs))
	}
	e := dom.NewElement(nil, tag,
		utils.None[dom.Namespace](), utils.None[string](), utils.None[string](),
		false, utils.None[string](), nil)
	for i := 0; i < len(attrs); i += 2 {
		e.SetAttribute(attrs[i], attrs[i+1])
	}
	return e
}

// appendKids appends each child to parent (preserving order) and returns parent.
func appendKids(parent dom.ElementNode, kids ...dom.Node) dom.ElementNode {
	for _, k := range kids {
		parent.AppendChild(k)
	}
	return parent
}

// mustMatch parses sel and reports whether it matches node.
func mustMatch(t *testing.T, sel string, node dom.ElementNode) bool {
	t.Helper()
	s, err := selector.NewSelector(sel)
	if err != nil {
		t.Fatalf("NewSelector(%q): unexpected error: %v", sel, err)
	}
	return s.Matches(node)
}

func TestMatches_SingleCompound(t *testing.T) {
	el := newEl(t, "div", "class", "a c", "id", "x")
	cases := []struct {
		sel  string
		want bool
	}{
		{"div", true},
		{"span", false},
		{".a", true},
		{".c", true},
		{".b", false},
		{"#x", true},
		{"#y", false},
		{"div.a#x", true},
		{".a.c", true}, // all simple selectors must match (AND)
		{".a.d", false},
		{"*", true},
	}
	for _, c := range cases {
		if got := mustMatch(t, c.sel, el); got != c.want {
			t.Errorf("Matches(%q) = %v, want %v", c.sel, got, c.want)
		}
	}
}

func TestMatches_Descendant(t *testing.T) {
	deep := newEl(t, "span", "class", "b")
	appendKids(newEl(t, "div", "class", "a"),
		appendKids(newEl(t, "div"), deep)) // .a > div > .b

	if !mustMatch(t, ".a .b", deep) {
		t.Errorf(".a .b should match a nested descendant")
	}
	orphan := newEl(t, "span", "class", "b")
	if mustMatch(t, ".a .b", orphan) {
		t.Errorf(".a .b should not match without an .a ancestor")
	}
}

func TestMatches_Child(t *testing.T) {
	direct := newEl(t, "span", "class", "b")
	appendKids(newEl(t, "div", "class", "a"), direct) // .a > .b

	grand := newEl(t, "span", "class", "b")
	appendKids(newEl(t, "div", "class", "a"),
		appendKids(newEl(t, "div"), grand)) // .a > div > .b

	if !mustMatch(t, ".a > .b", direct) {
		t.Errorf(".a > .b should match a direct child")
	}
	if mustMatch(t, ".a > .b", grand) {
		t.Errorf(".a > .b should not match a grandchild")
	}
	if !mustMatch(t, ".a .b", grand) {
		t.Errorf(".a .b should match the grandchild as a descendant")
	}
}

func TestMatches_Backtracking(t *testing.T) {
	// .a > bOuter(.b) > wrap > bInner(.b) > c(.c)
	// Greedy-nearest picks bInner for ".b", whose parent (wrap) is not .a, so the
	// child combinator dead-ends. Matching must backtrack to bOuter, whose parent
	// is .a, to succeed.
	c := newEl(t, "span", "class", "c")
	appendKids(newEl(t, "div", "class", "a"),
		appendKids(newEl(t, "div", "class", "b"),
			appendKids(newEl(t, "div"),
				appendKids(newEl(t, "div", "class", "b"), c))))

	if !mustMatch(t, ".a > .b .c", c) {
		t.Errorf(".a > .b .c should match via backtracking to the outer .b")
	}
}

func TestMatches_Siblings(t *testing.T) {
	// parent > a(.a), <text>, mid(.m), b(.b)
	a := newEl(t, "i", "class", "a")
	mid := newEl(t, "i", "class", "m")
	b := newEl(t, "i", "class", "b")
	appendKids(newEl(t, "div"),
		a, dom.NewTextNode(nil, "ws", nil), mid, b)

	// '~' matches any preceding element sibling.
	if !mustMatch(t, ".a ~ .b", b) {
		t.Errorf(".a ~ .b should match a non-adjacent preceding sibling")
	}
	// '+' matches only the immediately preceding element sibling, which is .m.
	if mustMatch(t, ".a + .b", b) {
		t.Errorf(".a + .b should not match when .a is not the immediate sibling")
	}
	if !mustMatch(t, ".m + .b", b) {
		t.Errorf(".m + .b should match the immediately preceding element sibling")
	}

	// Adjacent across an interleaved text node: a(.a), <text>, b(.b).
	a2 := newEl(t, "i", "class", "a")
	b2 := newEl(t, "i", "class", "b")
	appendKids(newEl(t, "div"), a2, dom.NewTextNode(nil, "ws", nil), b2)
	if !mustMatch(t, ".a + .b", b2) {
		t.Errorf(".a + .b should treat .a as adjacent across a text node")
	}
}

func TestMatches_List(t *testing.T) {
	span := newEl(t, "span", "class", "b")
	if !mustMatch(t, "a, .b", span) {
		t.Errorf("a, .b should match via the .b alternative")
	}
	anchor := newEl(t, "a")
	if !mustMatch(t, "a, .b", anchor) {
		t.Errorf("a, .b should match via the a alternative")
	}
	p := newEl(t, "p")
	if mustMatch(t, "a, .b", p) {
		t.Errorf("a, .b should not match a <p> with no class")
	}
}

func TestParseTypeSelector_Namespace(t *testing.T) {
	// *|* -> any-namespace universal
	comp := firstCompound(t, "*|*")
	ts := comp.Type
	if ts == nil || !ts.HasNamespace || ts.Prefix != "*" || !ts.Universal {
		t.Fatalf("*|* type = %+v", ts)
	}

	// |div -> no-namespace type
	comp = firstCompound(t, "|div")
	ts = comp.Type
	if ts == nil || !ts.HasNamespace || ts.Prefix != "" || ts.LocalName != "div" {
		t.Fatalf("|div type = %+v", ts)
	}
}
