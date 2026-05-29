package cssom

import (
	"errors"

	css_parser "github.com/VisualSource/plex/internal/css/parser"
	css_tokenizer "github.com/VisualSource/plex/internal/css/tokenizer"
	"github.com/Zyko0/go-sdl3/sdl"
)

/*
https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/Properties/display
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

type Display struct {
	Outer string
	Inner string
}

func parseDisplay(tokens []css_tokenizer.Token) (Display, error) {
	if len(tokens) < 1 {
		return Display{}, errors.New("invalid property")
	}

	first := css_parser.GetTokenValueAsString(tokens[0])

	switch first {
	case "inline-block", "inline-table", "inline-flex", "inline-grid":
		if (len(tokens)) != 1 {
			return Display{}, errors.New("syntax error")
		}

		return Display{
			Outer: "block",
			Inner: "flow",
		}, nil

	case "none", "contents":
		if (len(tokens)) != 1 {
			return Display{}, errors.New("syntax error")
		}

		return Display{
			Outer: first,
		}, nil
	default:
		return Display{}, errors.New("syntax error")
	}
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
func parseSize(tokens []css_tokenizer.Token) (any, error) {

	return nil, errors.New("invalid css size value")
}

var unsetColor = sdl.FColor{}

/*
background-color,color

<color>
*/
func parseColor(tokens []css_tokenizer.Token) (sdl.FColor, error) {

	return unsetColor, errors.New("unknown css color")
}

func parseMargin(tokens []css_tokenizer.Token) (any, error) {

	return nil, ErrUnsupportedProperty
}
