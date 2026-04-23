package dom

type Node interface {
	isNode()
	Tag() string
	Namespace() Namespace
}

type DocumentType struct {
	Name     string
	PublicId string
	SystemId string
}

func (d DocumentType) isNode() {}
func (d DocumentType) Tag() string {
	return ""
}
func (d DocumentType) Namespace() Namespace {
	return NamespaceHTML
}

func NewDocumentType(name string, publicId string, systemId string) *DocumentType {
	return &DocumentType{Name: name, PublicId: publicId, SystemId: systemId}
}
