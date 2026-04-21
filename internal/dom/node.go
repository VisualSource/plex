package dom

type Node interface {
	isNode()
}

type DocumentType struct {
	Name     string
	PublicId string
	SystemId string
}

func (d DocumentType) isNode() {}

func NewDocumentType(name string, publicId string, systemId string) *DocumentType {
	return &DocumentType{Name: name, PublicId: publicId, SystemId: systemId}
}
