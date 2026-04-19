package html_parser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/dom"
)

type testCase struct {
	data             string
	errors           []string
	documentFragment string
	scripting        bool
	document         []string
}

func TestHtmlParser(t *testing.T) {
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

func validateTest(t *testing.T, doc *dom.Document, parser *HtmlParser, test *testCase) {
	t.Helper()
}
