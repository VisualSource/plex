package layout_test

import (
	"errors"
	"testing"

	"github.com/VisualSource/plex/internal/css/parser"
	"github.com/VisualSource/plex/internal/layout"
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
			_, err := layout.NewSelector(c.in)
			switch {
			case c.wantErr && !errors.Is(err, parser.ErrSyntax):
				t.Fatalf("NewSelector(%q): want ErrSyntax, got %v", c.in, err)
			case !c.wantErr && err != nil:
				t.Fatalf("NewSelector(%q): unexpected error: %v", c.in, err)
			}
		})
	}
}

func TestNewSelector_Structure(t *testing.T) {
	sel, err := layout.NewSelector("ul.menu > li#x")
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
	if cs.Units[0].Combinator != layout.CombinatorNone {
		t.Fatalf("unit[0] combinator = %v, want None", cs.Units[0].Combinator)
	}
	if cs.Units[1].Combinator != layout.CombinatorChild {
		t.Fatalf("unit[1] combinator = %v, want Child", cs.Units[1].Combinator)
	}
	if got := cs.Units[0].Compound.Type; got == nil || got.LocalName != "ul" {
		t.Fatalf("unit[0] type = %+v, want ul", got)
	}
	if got := cs.Units[1].Compound.Type; got == nil || got.LocalName != "li" {
		t.Fatalf("unit[1] type = %+v, want li", got)
	}

	// ul.menu => c=1,b=1 ; li#x => c=1,a=1 ; total a=1,b=1,c=2
	want := layout.Specificity{A: 1, B: 1, C: 2}
	if got := cs.Specificity(); got != want {
		t.Fatalf("specificity = %+v, want %+v", got, want)
	}
	if sel.Specificity != want.Value() {
		t.Fatalf("Selector.Specificity = %d, want %d", sel.Specificity, want.Value())
	}
}

func TestNewSelector_SpecificityMaxOverList(t *testing.T) {
	// "div, #id" -> max( c=1 , a=1 ) = a=1
	sel, err := layout.NewSelector("div, #id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := (layout.Specificity{A: 1}).Value(); sel.Specificity != want {
		t.Fatalf("Selector.Specificity = %d, want %d", sel.Specificity, want)
	}
}

func firstCompound(t *testing.T, in string) *layout.CompoundSelector {
	t.Helper()
	sel, err := layout.NewSelector(in)
	if err != nil {
		t.Fatalf("NewSelector(%q): unexpected error: %v", in, err)
	}
	return sel.List[0].Units[0].Compound
}

func TestParseAttributeSelector(t *testing.T) {
	cases := []struct {
		in       string
		matcher  layout.AttrMatcher
		value    string
		modifier layout.AttrModifier
	}{
		{"[href]", layout.AttrPresence, "", layout.AttrModNone},
		{`[href="x"]`, layout.AttrEquals, "x", layout.AttrModNone},
		{`[rel~="next"]`, layout.AttrIncludes, "next", layout.AttrModNone},
		{"[lang|=en]", layout.AttrDashMatch, "en", layout.AttrModNone},
		{`[href^="https"]`, layout.AttrPrefix, "https", layout.AttrModNone},
		{`[src$=".png"]`, layout.AttrSuffix, ".png", layout.AttrModNone},
		{`[title*="x"]`, layout.AttrSubstring, "x", layout.AttrModNone},
		{`[href="X" i]`, layout.AttrEquals, "X", layout.AttrModCaseInsensitive},
		{`[href="X" s]`, layout.AttrEquals, "X", layout.AttrModCaseSensitive},
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			comp := firstCompound(t, c.in)
			if len(comp.Subclass) != 1 {
				t.Fatalf("subclass count = %d, want 1", len(comp.Subclass))
			}
			attr, ok := comp.Subclass[0].(*layout.AttributeSelector)
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
	if pc, ok := comp.Subclass[0].(*layout.PseudoClassSelector); !ok || pc.Name != "hover" || pc.Functional {
		t.Fatalf(":hover = %+v", comp.Subclass[0])
	}

	// :before -> legacy pseudo-element in Pseudos
	comp = firstCompound(t, ":before")
	if len(comp.Pseudos) != 1 {
		t.Fatalf(":before pseudos count = %d, want 1", len(comp.Pseudos))
	}
	if pe, ok := comp.Pseudos[0].(*layout.PseudoElementSelector); !ok || pe.Name != "before" || !pe.Legacy {
		t.Fatalf(":before = %+v", comp.Pseudos[0])
	}

	// ::before -> modern pseudo-element
	comp = firstCompound(t, "::before")
	if pe, ok := comp.Pseudos[0].(*layout.PseudoElementSelector); !ok || pe.Name != "before" || pe.Legacy {
		t.Fatalf("::before = %+v", comp.Pseudos[0])
	}

	// :not(.x) -> functional pseudo-class with raw args
	comp = firstCompound(t, ":not(.x)")
	pc, ok := comp.Subclass[0].(*layout.PseudoClassSelector)
	if !ok || pc.Name != "not" || !pc.Functional {
		t.Fatalf(":not(.x) = %+v", comp.Subclass[0])
	}
	if len(pc.RawArgs) == 0 {
		t.Fatalf(":not(.x) raw args are empty")
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
