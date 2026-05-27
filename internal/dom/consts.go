package dom

import "github.com/VisualSource/plex/internal/utils"

type Namespace string
type NamespaceOption = utils.Option[Namespace]

const (
	NamespaceHTML   Namespace = "http://www.w3.org/1999/xhtml"
	NamespaceMathML Namespace = "http://www.w3.org/1998/Math/MathML"
	NamespaceSVG    Namespace = "http://www.w3.org/2000/svg"
	NamespaceXLink  Namespace = "http://www.w3.org/1999/xlink"
	NamespaceXML    Namespace = "http://www.w3.org/XML/1998/namespace"
	NamespaceXMLNS  Namespace = "http://www.w3.org/2000/xmlns/"
)

func PrefixToNamespace(prefix string) Namespace {
	switch prefix {
	case "svg":
		return NamespaceSVG
	case "math":
		return NamespaceMathML
	case "xlink":
		return NamespaceXLink
	case "xmlns":
		return NamespaceXMLNS
	case "xml":
		return NamespaceXML
	default:
		return NamespaceHTML
	}
}

func NamespaceToPrefix(namespace Namespace) string {
	switch namespace {
	case NamespaceHTML:
		return "html"
	case NamespaceMathML:
		return "math"
	case NamespaceSVG:
		return "svg"
	case NamespaceXLink:
		return "xlink"
	case NamespaceXML:
		return "xml"
	case NamespaceXMLNS:
		return "xmlns"
	default:
		return ""
	}
}

type QuirksMode string

const (
	QuirksMode_Limited  QuirksMode = "limited-quirks"
	QuirksMode_Quirks   QuirksMode = "quirks"
	QuirksMode_NoQuirks QuirksMode = "no-quirks"
)
