package cssom

import (
	"errors"

	css_parser "github.com/VisualSource/plex/internal/css/parser"
	css_tokenizer "github.com/VisualSource/plex/internal/css/tokenizer"
)

/*
https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/Properties/display
https://drafts.csswg.org/css-display/#the-display-properties

display =

	[ <display-outside> || <display-inside> ]         |
	<display-listitem>                                |
	<display-internal>                                |
	<display-box>                                     |
	<display-legacy>                                  |
	grid-lanes                                        |
	inline-grid-lanes                                 |
	<display-outside> || [ <display-inside> | math ]

<display-outside> =

	block   |
	inline  |
	run-in

<display-inside> =

	flow       |
	flow-root  |
	table      |
	flex       |
	grid       |
	ruby

<display-listitem> =

	<display-outside>?     &&
	[ flow | flow-root ]?  &&
	list-item

<display-internal> =

	table-row-group      |
	table-header-group   |
	table-footer-group   |
	table-row            |
	table-cell           |
	table-column-group   |
	table-column         |
	table-caption        |
	ruby-base            |
	ruby-text            |
	ruby-base-container  |
	ruby-text-container

<display-box> =

	contents  |
	none

<display-legacy> =

	inline-block  |
	inline-table  |
	inline-flex   |
	inline-grid
*/
func parseDisplay(tokens []css_tokenizer.Token) (any, error) {
	items := nonWS(tokens)

	if len(items) < 1 {
		return Display{}, errors.New("invalid property")
	}

	first := css_parser.GetTokenValueAsString(items[0])

	// <display-outline> <display-inside> = (block | inline | run-in) | (flow | flow-root | table | flex | grid | ruby)
	// <display-outline> = block | inline | run-in
	// <display-inside> = flow | flow-root | table | flex | grid | ruby

	// <display-listitem> = Optional<(block | inline | run-in)> && Optional<"flow"|"flow-root"> && list-item

	switch first {
	case "block", "inline", "run-in":

		if len(items) >= 2 {
			second := css_parser.GetTokenValueAsString(items[1])

			switch second {
			case "flow", "flow-root":
				if len(items) == 3 {
					third := css_parser.GetTokenValueAsString(items[2])
					if third != "list-item" {
						return Display{}, ErrInvalidCssValue
					}

					return Display{
						Outer:    first,
						Inner:    second,
						ListItem: true,
					}, nil
				}
				fallthrough
			case "table", "flex", "grid", "ruby":
				if len(items) == 2 {
					return Display{
						Outer: first,
						Inner: second,
					}, nil
				}
				fallthrough
			default:
				return Display{}, ErrInvalidCssValue
			}
		}

		// len(items) >= 2
		//    <display-inside>
		//    if <display-inside> == "flow" | "flow-root" && len(items) == 3 && item[2] == "list-item"
		//            return <display-listitem>
		//    return <display-outside> <display-inside>

		// <display-outline>

		// block -> block flow
		// inline -> inline flow
		// run-in -> run-in flow
		return newSingleDisplay(items, first, "flow", false)
	case "flow", "flow-root", "table", "flex", "grid", "ruby":

		if first == "flow" || first == "flow-root" && len(items) > 1 {
			second := css_parser.GetTokenValueAsString(items[1])
			if second != "list-item" {
				return Display{}, ErrInvalidCssValue
			}
			return Display{
				Outer:    "block",
				Inner:    first,
				ListItem: true,
			}, nil
		}

		// flow      -> block flow
		// flow-root -> block flow-root
		// table     -> block table
		// flex      -> block flex
		// grid      -> block grid
		// ruby      -> inline ruby
		outer := "block"
		if first == "ruby" {
			outer = "inline"
		}

		return newSingleDisplay(items, outer, first, false)
	case "list-item":
		return newSingleDisplay(items, "block", "flow", true)
	//#region <display-internal>
	case "table-row-group", "table-header-group", "table-footer-group", "table-row", "table-cell", "table-column-group", "table-column",
		"table-caption", "ruby-base", "ruby-text", "ruby-base-container", "ruby-text-container": //<display-internal>
		return newSingleDisplay(items, first, first, false)
	//#endregion

	//#region <display-box>
	case "none", "contents":
		return newSingleDisplay(items, first, first, false)
	//#endregion

	//#region <display-legacy>
	case "inline-block":
		return newSingleDisplay(items, "inline", "flow-root", false)
	case "inline-table":
		return newSingleDisplay(items, "inline", "table", false)
	case "inline-flex":
		return newSingleDisplay(items, "inline", "flex", false)
	case "inline-grid":
		return newSingleDisplay(items, "inline", "grid", false)
	//#endregion

	default:
		return Display{}, ErrInvalidCssValue
	}
}

type Display struct {
	Outer    string
	Inner    string
	ListItem bool
}

func newSingleDisplay(items []css_tokenizer.Token, outer string, inner string, listItem bool) (Display, error) {
	if len(items) != 1 {
		return Display{}, ErrInvalidCssValue
	}

	return Display{
		Outer:    outer,
		Inner:    inner,
		ListItem: listItem,
	}, nil
}
