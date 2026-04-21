package dom

type QuirksMode string

const (
	QuirksMode_Limited  QuirksMode = "limited-quirks"
	QuirksMode_Quirks   QuirksMode = "quirks"
	QuirksMode_NoQuirks QuirksMode = "no-quirks"
)

type Document struct {
	children       []Node
	QuirksMode     QuirksMode
	ParserNoChangeMode bool
}

func (d *Document) AppendChild(node Node) {
	d.children = append(d.children, node)
}


func (d Document) IsIframeSrcDoc() bool {
	return false
}