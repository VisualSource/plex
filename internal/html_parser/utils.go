package html_parser

// https://html.spec.whatwg.org/multipage/parsing.html#html-integration-point
func isHTMLIntergrationPoint(node *dom.Node) bool {
	return false
}

// https://html.spec.whatwg.org/multipage/parsing.html#mathml-text-integration-point
func isMathMLIntegrationPoint(node *dom.Node) bool {

	return false
}