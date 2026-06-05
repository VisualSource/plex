package selector

import "github.com/VisualSource/plex/internal/dom"

func QuerySelector(node dom.Node, selector string) (dom.Node, error) {
	item, err := NewSelector(selector)
	if err != nil {
		return nil, err
	}

	return walkFirst(node, item), nil
}

func QuerySelectorAll(node dom.Node, selector string) ([]dom.Node, error) {
	item, err := NewSelector(selector)
	if err != nil {
		return nil, err
	}

	var results []dom.Node
	walkAll(node, item, &results)
	return results, nil
}

func walkFirst(node dom.Node, sel *Selector) dom.Node {
	for _, child := range node.Children() {
		if el, ok := child.(dom.ElementNode); ok {
			if sel.Matches(el) {
				return child
			}
		}
		if found := walkFirst(child, sel); found != nil {
			return found
		}
	}
	return nil
}

func walkAll(node dom.Node, sel *Selector, out *[]dom.Node) {
	for _, child := range node.Children() {
		if el, ok := child.(dom.ElementNode); ok {
			if sel.Matches(el) {
				*out = append(*out, child)
			}
		}
		walkAll(child, sel, out)
	}
}
