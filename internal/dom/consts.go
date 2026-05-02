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

type QuirksMode string

const (
	QuirksMode_Limited  QuirksMode = "limited-quirks"
	QuirksMode_Quirks   QuirksMode = "quirks"
	QuirksMode_NoQuirks QuirksMode = "no-quirks"
)
