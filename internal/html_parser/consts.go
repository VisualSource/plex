package html_parser

type InsertionMode uint

const (
	mode_Initial InsertionMode = iota
	mode_BeforeHtml
	mode_BeforeHead
	mode_InHead
	mode_InHeadNoScript
	mode_AfterHead
	mode_InBody
	mode_Text
	mode_InTable
	mode_InTableText
	mode_InCaption
	mode_InColumnGroup
	mode_InTableBody
	mode_InRow
	mode_InCell
	mode_InTemplate
	mode_AfterBody
	mode_InFrameset
	mode_AfterFrameset
	mode_AfterAfterBody
	mode_AfterAfterFrameset
)

type ScriptingMode uint

const (
	// Scripts are processed when inserted, respecting async and defer attributes and blocking the parser when encountering a classic script.
	mode_Normal ScriptingMode = iota
	// Scripts are disabled, and the noscript element can represent fallback content.
	mode_Disabled
	// Scripts are enabled, however they are marked as already started, essentially preventing them from executing.
	// This is the default mode of the HTML fragment parsing algorithm.
	mode_Inert
	// Scripts are executed as soon as they are inserted into the document as part of a the HTML fragment parsing algorithm,
	// ignoring async and defer attributes. This mode is used by createContextualFragment().
	mode_Fragment
)

type Namespace string

const (
	NamespaceHTML   Namespace = "http://www.w3.org/1999/xhtml"
	NamespaceMathML           = "http://www.w3.org/1998/Math/MathML"
	NamespaceSVG              = "http://www.w3.org/2000/svg"
	NamespaceXLink            = "http://www.w3.org/1999/xlink"
	NamespaceXML              = "http://www.w3.org/XML/1998/namespace"
	NamespaceXMLNS            = "http://www.w3.org/2000/xmlns/"
)
