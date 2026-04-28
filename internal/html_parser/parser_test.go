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

func newMock(tag string, parent dom.Node) *mockNode {
	return &mockNode{tag: tag, parent: parent, namespace: dom.NamespaceHTML}
}

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

	testSubset, _ := os.LookupEnv("TEST_HTML_PARSER_SUBSET")
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
		scanner.Split(bufio.ScanLines)

		testCases := make([]testCase, 0)

		docLines := []string{}

		currentTestCase := testCase{}
		mode := 0
		for scanner.Scan() {
			line := scanner.Text()

			switch line {
			case "#data":
				mode = 0
				continue
			case "#document":
				mode = 1
				continue
			case "#errors", "#new-errors":
				mode = 2
				continue
			case "#document-fragment":
				mode = 3
				continue
			case "#script-off":
				currentTestCase.scripting = false
				continue
			case "#script-on":
				currentTestCase.scripting = true
				continue
			case "":
				currentTestCase.document = docLines
				testCases = append(testCases, currentTestCase)
				currentTestCase = testCase{}
				docLines = []string{}
				mode = 0
				continue
			default:
			}

			switch mode {
			case 0:
				currentTestCase.data = line
			case 1:
				doc, found := strings.CutPrefix(line, "| ")
				if found {
					docLines = append(docLines, doc)
				}
			case 2:
				currentTestCase.errors = append(currentTestCase.errors, line)
			case 3:
				currentTestCase.documentFragment = line
			}

		}

		tests[testname] = testCases
	}

	return tests
}

func printTree(root dom.Node, ident int) []string {
	var output []string

	for _, node := range root.Children() {

		switch tag := node.(type) {
		case *dom.Text:
			output = append(output, fmt.Sprintf("%s\"<%s\"", strings.Repeat(" ", ident), tag.Data))
		case *dom.Comment:
			output = append(output, fmt.Sprintf("%s<!-- %s -->", strings.Repeat(" ", ident), tag.Data))
		case *dom.Element:
			output = append(output, fmt.Sprintf("%s<%s>", strings.Repeat(" ", ident), tag.Tag()))

			if children := node.Children(); children != nil {
				tree := printTree(node, ident+2)
				output = append(output, tree...)
			}
		case *dom.TemplateElement:
			output = append(output, fmt.Sprintf("%s<%s>", strings.Repeat(" ", ident), tag.Tag()))

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
		t.Fatalf("was expecting %#v but was given %#v", testCase.document, tree)
	}
}
