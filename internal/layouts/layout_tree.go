package layouts

import (
	"strings"
	"unicode"

	"github.com/VisualSource/plex/internal/css/cssom"
	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/layouts/styletree"
)

func NewLayoutTree(node *styletree.StyledNode, ctx *Context, width, height float64) *Box {

	dim := &Dimensions{
		Content: Rect{
			H: 0, //height, //TODO: need stacking context or something
			W: width,
		},
	}

	box := buildLayoutTree(node)
	box.calculateDimensions(dim, ctx)

	return box
}

func buildLayoutTree(node *styletree.StyledNode) *Box {

	outer, inner := getDisplayValue(styletree.GetProp[cssom.Display](node.SpecifiedValues, "display"))

	box := &Box{
		Style:     node,
		OuterType: outer,
		InnerType: inner,
		Children:  make([]*Box, 0),
	}

	for _, info := range processChildren(node.Children) {
		if info.isText {
			textBox := &Box{
				Style:       info.node,
				OuterType:   OuterBoxType_Inline,
				InnerType:   InnerBoxType_Flow,
				Children:    make([]*Box, 0),
				TextContent: info.text,
			}
			container := box.getInlineContainer()
			container.Children = append(container.Children, textBox)
			continue
		}

		switch info.outer {
		case OuterBoxType_Block:
			b := buildLayoutTree(info.node)
			box.Children = append(box.Children, b)
		case OuterBoxType_Inline:
			container := box.getInlineContainer()
			b := buildLayoutTree(info.node)
			container.Children = append(container.Children, b)
		}
	}

	return box
}

type childInfo struct {
	node   *styletree.StyledNode
	isText bool
	outer  OuterBoxType // for non-text only
	text   string       // normalized text (for text only)
}

// processChildren classifies child styled nodes and applies CSS-style
// whitespace processing for white-space: normal:
//   - whitespace-only text nodes between block boundaries are dropped
//   - kept text nodes have internal whitespace collapsed to single spaces
//   - leading whitespace is stripped at the start of an inline run; trailing
//     whitespace is stripped at the end of an inline run
//
// Properties like `white-space: pre` are not yet honored.
func processChildren(children []*styletree.StyledNode) []childInfo {
	type rawEntry struct {
		node     *styletree.StyledNode
		isText   bool
		outer    OuterBoxType
		raw      string
		wsOnly   bool
		isInline bool // text or inline element
	}

	raws := make([]rawEntry, 0, len(children))
	for _, child := range children {
		if textNode, ok := child.Element.(*dom.Text); ok {
			raws = append(raws, rawEntry{
				node:     child,
				isText:   true,
				raw:      textNode.Data,
				wsOnly:   strings.TrimSpace(textNode.Data) == "",
				isInline: true,
			})
			continue
		}
		outer, _ := getDisplayValue(styletree.GetProp[cssom.Display](child.SpecifiedValues, "display"))
		raws = append(raws, rawEntry{
			node:     child,
			outer:    outer,
			isInline: outer == OuterBoxType_Inline,
		})
	}

	prevIsInline := func(i int) bool {
		for j := i - 1; j >= 0; j-- {
			if raws[j].isText && raws[j].wsOnly {
				continue
			}
			return raws[j].isInline
		}
		return false
	}
	nextIsInline := func(i int) bool {
		for j := i + 1; j < len(raws); j++ {
			if raws[j].isText && raws[j].wsOnly {
				continue
			}
			return raws[j].isInline
		}
		return false
	}

	result := make([]childInfo, 0, len(raws))
	for i, r := range raws {
		if !r.isText {
			result = append(result, childInfo{node: r.node, outer: r.outer})
			continue
		}

		collapsed := collapseWhitespace(r.raw)
		if !prevIsInline(i) {
			collapsed = strings.TrimLeft(collapsed, " ")
		}
		if !nextIsInline(i) {
			collapsed = strings.TrimRight(collapsed, " ")
		}
		if collapsed == "" {
			continue
		}
		result = append(result, childInfo{node: r.node, isText: true, text: collapsed})
	}
	return result
}

func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inWS := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !inWS {
				b.WriteByte(' ')
				inWS = true
			}
			continue
		}
		b.WriteRune(r)
		inWS = false
	}
	return b.String()
}
