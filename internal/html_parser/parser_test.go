package html_parser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/html_tokenizer"
	"github.com/kr/pretty"
)

type testCase struct {
	data             string
	errors           []string
	documentFragment string
	scripting        bool
	document         []string
}

type mockNode struct {
	tag       string
	parent    dom.Node
	namespace dom.Namespace
}

func (m *mockNode) IsNode() uint                             { return 4 }
func (m *mockNode) Tag() string                              { return m.tag }
func (m *mockNode) Namespace() dom.Namespace                 { return m.namespace }
func (m *mockNode) AppendChild(dom.Node)                     {}
func (m *mockNode) PrependChild(dom.Node)                    {}
func (m *mockNode) InsertBefore(node dom.Node, ref dom.Node) {}
func (m *mockNode) Parent() dom.Node                         { return m.parent }
func (m *mockNode) Document() *dom.Document                  { return nil }
func (m *mockNode) PreviousSibling() dom.Node                { return nil }
func (m *mockNode) Children() []dom.Node                     { return nil }
func (m *mockNode) Remove()                                  {}
func (m *mockNode) RemoveChild(dom.Node) dom.Node            { return nil }

func newMock(tag string, parent dom.Node) *mockNode {
	return &mockNode{tag: tag, parent: parent, namespace: dom.NamespaceHTML}
}

type mockElementNode struct {
	mockNode
	attrs []dom.Attribute
}

func newMockElement(tag string) *mockElementNode {
	return &mockElementNode{mockNode: mockNode{tag: tag, namespace: dom.NamespaceHTML}}
}

func (m *mockElementNode) SetAttribute(key, value string) {}
func (m *mockElementNode) GetAttribute(key string) *dom.Attribute {
	for i := range m.attrs {
		if m.attrs[i].LocalName == key {
			return &m.attrs[i]
		}
	}
	return nil
}
func (m *mockElementNode) GetAttributeNS(ns dom.Namespace, localName string) *dom.Attribute {
	for i := range m.attrs {
		if m.attrs[i].NamespaceUri == ns && m.attrs[i].LocalName == localName {
			return &m.attrs[i]
		}
	}
	return nil
}
func (m *mockElementNode) HasAttribute(key string) bool { return m.GetAttribute(key) != nil }
func (m *mockElementNode) Attributes() []dom.Attribute  { return m.attrs }

func TestHtmlParser_appropriatePlaceForInsertingNode(t *testing.T) {
	newParser := func() *HtmlParser { return NewHtmlParser(strings.NewReader("")) }
	// check asserts Parent and Before on the returned InsertionLocation.
	// Pass nil for wantBefore when expecting an append (no reference node).
	check := func(t *testing.T, parent dom.Node, before dom.Node, wantParent dom.Node, wantBefore dom.Node) {
		t.Helper()
		if parent != wantParent {
			t.Errorf("Parent = %v, want %v", parent, wantParent)
		}
		if before != wantBefore {
			t.Errorf("Before = %v, want %v", before, wantBefore)
		}
	}

	t.Run("returns current node when no override and parent is nil", func(t *testing.T) {
		div := newMock("div", nil)
		p := newParser()
		p.openElementsStack = []dom.Node{div}
		parent, before := p.appropriatePlaceForInsertingNode(nil)
		check(t, parent, before, div, nil)
	})

	t.Run("returns current node when parent is a regular element", func(t *testing.T) {
		body := newMock("body", nil)
		div := newMock("div", body)
		p := newParser()
		p.openElementsStack = []dom.Node{body, div}
		parent, before := p.appropriatePlaceForInsertingNode(nil)
		check(t, parent, before, div, nil)
	})

	t.Run("returns adjusted when it is itself a TemplateElement", func(t *testing.T) {
		// spec step 3: if adjusted IS a template element, insert into its template contents;
		// TemplateElement.AppendChild already routes to templateContents, so returning adjusted is correct
		templateEl := &dom.TemplateElement{}
		p := newParser()
		p.openElementsStack = []dom.Node{templateEl}
		parent, before := p.appropriatePlaceForInsertingNode(nil)
		check(t, parent, before, templateEl, nil)
	})

	t.Run("returns adjusted even when its parent is a TemplateElement", func(t *testing.T) {
		// spec step 3 checks if adjusted itself is a template element, not its parent
		templateEl := &dom.TemplateElement{}
		div := newMock("div", templateEl)
		p := newParser()
		p.openElementsStack = []dom.Node{div}
		parent, before := p.appropriatePlaceForInsertingNode(nil)
		check(t, parent, before, div, nil)
	})

	t.Run("uses override target instead of current node", func(t *testing.T) {
		div := newMock("div", nil)
		span := newMock("span", nil)
		p := newParser()
		p.openElementsStack = []dom.Node{div}
		parent, before := p.appropriatePlaceForInsertingNode(span)
		check(t, parent, before, span, nil)
	})

	t.Run("returns override target even when its parent is a TemplateElement", func(t *testing.T) {
		// spec step 3 checks if adjusted itself is a template element, not its parent
		templateEl := &dom.TemplateElement{}
		div := newMock("div", nil)
		span := newMock("span", templateEl)
		p := newParser()
		p.openElementsStack = []dom.Node{div}
		parent, before := p.appropriatePlaceForInsertingNode(span)
		check(t, parent, before, span, nil)
	})

	t.Run("foster parenting skipped when target tag is not a table-related element", func(t *testing.T) {
		div := newMock("div", nil)
		p := newParser()
		p.openElementsStack = []dom.Node{div}
		p.fosterParenting = true
		parent, before := p.appropriatePlaceForInsertingNode(nil)
		check(t, parent, before, div, nil)
	})

	t.Run("foster parenting: template after table wins, returns template", func(t *testing.T) {
		div := newMock("div", nil)
		table := newMock("table", nil)
		tmpl := newMock("template", nil)
		p := newParser()
		// tempIdx(2) > tableIdx(1) → template wins
		p.openElementsStack = []dom.Node{div, table, tmpl}
		p.fosterParenting = true
		parent, before := p.appropriatePlaceForInsertingNode(newMock("table", nil))
		check(t, parent, before, tmpl, nil)
	})

	t.Run("foster parenting: template in stack with no table returns template", func(t *testing.T) {
		// spec: lastTemplate != nil && (no table OR template more recently added) → return template
		// condition should be (tableIdx == -1 || tempIdx > tableIdx), not (tableIdx != -1 && tempIdx > tableIdx)
		div := newMock("div", nil)
		tmpl := newMock("template", nil)
		p := newParser()
		p.openElementsStack = []dom.Node{div, tmpl}
		p.fosterParenting = true
		parent, before := p.appropriatePlaceForInsertingNode(newMock("tbody", nil))
		check(t, parent, before, tmpl, nil)
	})

	t.Run("foster parenting: no table and no template returns first open element", func(t *testing.T) {
		div := newMock("div", nil)
		p := newParser()
		p.openElementsStack = []dom.Node{div}
		p.fosterParenting = true
		parent, before := p.appropriatePlaceForInsertingNode(newMock("tbody", nil))
		check(t, parent, before, div, nil)
	})

	t.Run("foster parenting: table with parent sets Before=table so node lands just before it", func(t *testing.T) {
		tableParent := newMock("body", nil)
		div := newMock("div", nil)
		table := newMock("table", tableParent)
		p := newParser()
		p.openElementsStack = []dom.Node{div, table}
		p.fosterParenting = true
		parent, before := p.appropriatePlaceForInsertingNode(newMock("tr", nil))
		check(t, parent, before, tableParent, table)
	})

	t.Run("foster parenting: table without parent returns element before table in stack", func(t *testing.T) {
		div := newMock("div", nil)
		table := newMock("table", nil)
		p := newParser()
		p.openElementsStack = []dom.Node{div, table}
		p.fosterParenting = true
		parent, before := p.appropriatePlaceForInsertingNode(newMock("tr", nil))
		check(t, parent, before, div, nil)
	})
}

func TestHtmlParser_createElement(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		stream io.Reader
		// Named input parameters for target function.
		token          html_tokenizer.TokenTag
		namespace      dom.Namespace
		intendedParent dom.Node
		want           dom.Node
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewHtmlParser(tt.stream)
			got := p.createElement(tt.token, tt.namespace, tt.intendedParent)
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("createElement() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHtmlParser_insertElement(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		stream io.Reader
		// Named input parameters for target function.
		element dom.Node
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewHtmlParser(tt.stream)
			p.insertElement(tt.element)
		})
	}
}

func TestHtmlParser_insertForeginElement(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		stream io.Reader
		// Named input parameters for target function.
		token                 html_tokenizer.TokenTag
		namespace             dom.Namespace
		onlyAddToElementStack bool
		want                  dom.Node
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewHtmlParser(tt.stream)
			got := p.insertForeignElement(tt.token, tt.namespace, tt.onlyAddToElementStack)
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("insertForeginElement() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHtmlParser_insertCharacter(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		stream io.Reader
		// Named input parameters for target function.
		value rune
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewHtmlParser(tt.stream)
			p.insertCharacter(tt.value)
		})
	}
}

func TestHtmlParser_Parse(t *testing.T) {
	files := loadTestCases(t)

	for filename, tests := range files {
		for idx, tt := range tests {
			testname := fmt.Sprintf("[%s]: test %d", filename, idx)

			input := strings.NewReader(tt.data)
			t.Run(testname, func(t *testing.T) {
				if tt.documentFragment != "" || tt.scripting {
					t.Skipf("Skipping test '%s;' due to Fragment: %v, scripting: %v", testname, tt.documentFragment != "", tt.scripting)
					return
				}

				parser := NewHtmlParser(input)
				document, err := parser.Parse()
				if err != nil {
					t.Fatal(err)
				}

				validateTest(t, document, parser, &tt)
			})
		}
	}
}

func loadTestCases(t *testing.T) map[string][]testCase {
	t.Helper()

	testSubset := "tests3" //, _ := os.LookupEnv("TEST_HTML_PARSER_SUBSET")
	wantTestSubset := strings.Split(testSubset, ",")
	useSubset := testSubset != "" && len(wantTestSubset) != 0

	paths, err := filepath.Glob(filepath.Join("testdata", "*.test"))
	if err != nil {
		t.Fatal(err)
	}

	tests := make(map[string][]testCase)

	for _, path := range paths {
		_, filename := filepath.Split(path)
		testname := filename[:len(filename)-len(filepath.Ext(path))]

		if useSubset && !slices.Contains(wantTestSubset, testname) {
			t.Logf("ignore test file '%s'", testname)
			continue
		}

		t.Logf("using test file '%s'", testname)

		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		scanner.Split(bufio.ScanLines)

		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			t.Fatal(err)
		}

		// Split lines into per-test blocks. A blank line is a test separator
		// only when it is followed by "#data" (the next test's start) or EOF;
		// blank lines embedded in #data content or inside multi-line text
		// nodes in #document are part of the test, not separators.
		var blocks [][]string
		var current []string
		for i, line := range lines {
			isSeparator := line == "" && (i+1 == len(lines) || lines[i+1] == "#data")
			if isSeparator {
				if len(current) > 0 {
					blocks = append(blocks, current)
					current = nil
				}
				continue
			}
			current = append(current, line)
		}
		if len(current) > 0 {
			blocks = append(blocks, current)
		}

		testCases := make([]testCase, 0, len(blocks))
		for _, block := range blocks {
			var tc testCase
			var dataLines, docLines []string
			section := ""
			for _, line := range block {
				switch line {
				case "#data":
					section = "data"
					continue
				case "#errors", "#new-errors":
					section = "errors"
					continue
				case "#document":
					section = "document"
					continue
				case "#document-fragment":
					section = "document-fragment"
					continue
				case "#script-off":
					tc.scripting = false
					continue
				case "#script-on":
					tc.scripting = true
					continue
				}

				switch section {
				case "data":
					dataLines = append(dataLines, line)
				case "errors":
					tc.errors = append(tc.errors, line)
				case "document":
					if doc, ok := strings.CutPrefix(line, "| "); ok {
						docLines = append(docLines, doc)
					} else if len(docLines) > 0 {
						// Continuation of a text node that contains a newline.
						docLines[len(docLines)-1] += "\n" + line
					}
				case "document-fragment":
					tc.documentFragment = line
				}
			}
			tc.data = strings.Join(dataLines, "\n")
			tc.document = docLines
			testCases = append(testCases, tc)
		}

		tests[testname] = testCases
	}

	return tests
}

func printTree(root dom.Node, ident int) []string {
	var output []string

	for _, node := range root.Children() {

		switch tag := node.(type) {
		case *dom.DocumentType:
			ids := ""
			if tag.PublicId != "" || tag.SystemId != "" {
				ids = fmt.Sprintf(" \"%s\" \"%s\"", tag.PublicId, tag.SystemId)
			}

			output = append(output,
				fmt.Sprintf("%s<!DOCTYPE %s%s>", strings.Repeat(" ", ident), tag.Name, ids),
			)
		case *dom.Text:
			output = append(output, fmt.Sprintf("%s\"%s\"", strings.Repeat(" ", ident), tag.Data))
		case *dom.Comment:
			output = append(output, fmt.Sprintf("%s<!-- %s -->", strings.Repeat(" ", ident), tag.Data))
		case *dom.Element:
			output = append(output, fmt.Sprintf("%s<%s>", strings.Repeat(" ", ident), tag.Tag()))

			for _, attr := range tag.Attributes() {
				var localName string
				if attr.Prefix.IsSome() {
					localName = *attr.Prefix.Value + ":" + attr.LocalName
				} else {
					localName = attr.LocalName
				}
				output = append(output, fmt.Sprintf("%s%s=\"%s\"", strings.Repeat(" ", ident+2), localName, attr.Value))
			}

			if children := node.Children(); children != nil {
				tree := printTree(node, ident+2)
				output = append(output, tree...)
			}
		case *dom.TemplateElement:
			output = append(output, fmt.Sprintf("%s<%s>", strings.Repeat(" ", ident), tag.Tag()))
			for _, attr := range tag.Attributes() {
				var localName string
				if attr.Prefix.IsNone() {
					localName = *attr.Prefix.Value + ":" + attr.LocalName
				} else {
					localName = attr.LocalName
				}
				output = append(output, fmt.Sprintf("%s%s=\"%s\"", strings.Repeat(" ", ident+2), localName, attr.Value))
			}
			if children := node.Children(); children != nil {
				tree := printTree(node, ident+2)
				output = append(output, tree...)
			}
		}
	}

	return output
}

func validateTest(t *testing.T, document *dom.Document, _ *HtmlParser, testCase *testCase) {
	t.Helper()

	tree := printTree(document, 0)

	if !slices.Equal(testCase.document, tree) {
		diff := pretty.Diff(testCase.document, tree)
		t.Fatalf("tree mismatch:\nInput:%s\n%s\n\n(-expected +got):\n%s",
			testCase.data,
			strings.Join(diff, "\n"),
			unifiedDiff(testCase.document, tree),
		)
	}
}

func unifiedDiff(expected, got []string) string {
	const (
		ansiReset = "\x1b[0m"
		ansiRed   = "\x1b[31m"
		ansiGreen = "\x1b[32m"
		ansiDim   = "\x1b[2m"
	)

	n, m := len(expected), len(got)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if expected[i] == got[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	var lines []string
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case expected[i] == got[j]:
			lines = append(lines, ansiDim+"  "+expected[i]+ansiReset)
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			lines = append(lines, ansiGreen+"- "+expected[i]+ansiReset)
			i++
		default:
			lines = append(lines, ansiRed+"+ "+got[j]+ansiReset)
			j++
		}
	}
	for ; i < n; i++ {
		lines = append(lines, ansiGreen+"- "+expected[i]+ansiReset)
	}
	for ; j < m; j++ {
		lines = append(lines, ansiRed+"+ "+got[j]+ansiReset)
	}
	return strings.Join(lines, "\n")
}

func TestHtmlParser_reconstructActiveFormattingElements(t *testing.T) {
	newParser := func() *HtmlParser { return NewHtmlParser(strings.NewReader("")) }

	t.Run("no-op when list is empty", func(t *testing.T) {
		p := newParser()
		p.reconstructActiveFormattingElements()
		if len(p.activeFormattingElements) != 0 {
			t.Errorf("expected empty list, got %d entries", len(p.activeFormattingElements))
		}
	})

	t.Run("no-op when last entry is a marker", func(t *testing.T) {
		p := newParser()
		el := newMock("b", nil)
		p.activeFormattingElements = []activeFormattingItem{
			{Element: el},
			{IsMarker: true},
		}
		p.reconstructActiveFormattingElements()
		if len(p.openElementsStack) != 0 {
			t.Errorf("expected open stack unchanged, got %d entries", len(p.openElementsStack))
		}
		if p.activeFormattingElements[0].Element != el {
			t.Error("AFE entries should be unchanged")
		}
	})

	t.Run("no-op when last entry is already in open stack", func(t *testing.T) {
		p := newParser()
		el := newMock("b", nil)
		p.openElementsStack = []dom.Node{el}
		p.activeFormattingElements = []activeFormattingItem{{Element: el}}
		p.reconstructActiveFormattingElements()
		if len(p.openElementsStack) != 1 {
			t.Errorf("expected open stack unchanged, got %d entries", len(p.openElementsStack))
		}
		if p.activeFormattingElements[0].Element != el {
			t.Error("AFE entry should be unchanged")
		}
	})

	t.Run("single element is reconstructed", func(t *testing.T) {
		p := newParser()
		p.openElementsStack = []dom.Node{newMock("body", nil)}
		orig := newMockElement("b")
		p.activeFormattingElements = []activeFormattingItem{{Element: orig}}

		p.reconstructActiveFormattingElements()

		newEl := p.activeFormattingElements[0].Element
		if newEl == orig {
			t.Error("AFE entry should be replaced with a new element")
		}
		if newEl.Tag() != "b" {
			t.Errorf("expected tag %q, got %q", "b", newEl.Tag())
		}
		if !slices.Contains(p.openElementsStack, newEl) {
			t.Error("new element should be in open elements stack")
		}
	})

	t.Run("multiple consecutive elements all reconstructed", func(t *testing.T) {
		p := newParser()
		p.openElementsStack = []dom.Node{newMock("body", nil)}
		origB := newMockElement("b")
		origI := newMockElement("i")
		origU := newMockElement("u")
		p.activeFormattingElements = []activeFormattingItem{
			{Element: origB},
			{Element: origI},
			{Element: origU},
		}

		p.reconstructActiveFormattingElements()

		for idx, orig := range []*mockElementNode{origB, origI, origU} {
			newEl := p.activeFormattingElements[idx].Element
			if newEl == orig {
				t.Errorf("entry %d: should be replaced with a new element", idx)
			}
			if newEl.Tag() != orig.tag {
				t.Errorf("entry %d: expected tag %q, got %q", idx, orig.tag, newEl.Tag())
			}
			if !slices.Contains(p.openElementsStack, newEl) {
				t.Errorf("entry %d: new element not in open elements stack", idx)
			}
		}
	})

	t.Run("reconstruction stops at marker boundary", func(t *testing.T) {
		p := newParser()
		p.openElementsStack = []dom.Node{newMock("body", nil)}
		before := newMockElement("span")
		after := newMockElement("b")
		p.activeFormattingElements = []activeFormattingItem{
			{Element: before},
			{IsMarker: true},
			{Element: after},
		}

		p.reconstructActiveFormattingElements()

		if p.activeFormattingElements[0].Element != before {
			t.Error("element before marker should not be replaced")
		}
		newEl := p.activeFormattingElements[2].Element
		if newEl == after {
			t.Error("element after marker should be replaced")
		}
		if newEl.Tag() != "b" {
			t.Errorf("expected tag %q, got %q", "b", newEl.Tag())
		}
		if !slices.Contains(p.openElementsStack, newEl) {
			t.Error("new element should be in open elements stack")
		}
	})

	t.Run("reconstruction stops at open-stack element boundary", func(t *testing.T) {
		p := newParser()
		boundary := newMockElement("i")
		p.openElementsStack = []dom.Node{newMock("body", nil), boundary}
		before := newMockElement("span")
		after := newMockElement("b")
		p.activeFormattingElements = []activeFormattingItem{
			{Element: before},
			{Element: boundary},
			{Element: after},
		}

		p.reconstructActiveFormattingElements()

		if p.activeFormattingElements[0].Element != before {
			t.Error("element before boundary should not be replaced")
		}
		if p.activeFormattingElements[1].Element != boundary {
			t.Error("boundary element in open stack should not be replaced")
		}
		newEl := p.activeFormattingElements[2].Element
		if newEl == after {
			t.Error("element after boundary should be replaced")
		}
		if newEl.Tag() != "b" {
			t.Errorf("expected tag %q, got %q", "b", newEl.Tag())
		}
		if !slices.Contains(p.openElementsStack, newEl) {
			t.Error("new element should be in open elements stack")
		}
	})

	t.Run("attributes are preserved on reconstructed element", func(t *testing.T) {
		p := newParser()
		p.openElementsStack = []dom.Node{newMock("body", nil)}
		el := newMockElement("a")
		el.attrs = []dom.Attribute{
			{LocalName: "href", Value: "https://example.com"},
			{LocalName: "class", Value: "link"},
		}
		p.activeFormattingElements = []activeFormattingItem{{Element: el}}

		p.reconstructActiveFormattingElements()

		newEl, ok := p.activeFormattingElements[0].Element.(dom.ElementNode)
		if !ok {
			t.Fatal("new element does not implement ElementNode")
		}
		if attr := newEl.GetAttribute("href"); attr == nil || attr.Value != "https://example.com" {
			t.Errorf("href attribute not preserved: %v", attr)
		}
		if attr := newEl.GetAttribute("class"); attr == nil || attr.Value != "link" {
			t.Errorf("class attribute not preserved: %v", attr)
		}
	})
}
