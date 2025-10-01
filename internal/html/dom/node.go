package html

const (
	Node_Element               = 1
	Node_Attribute             = 2
	Node_Text                  = 3
	Node_CDATASection          = 4
	Node_ProcessingInstruction = 7
	Node_Comment               = 8
	Node_Document              = 9
	Node_DocumentType          = 10
	Node_DocumentFragment      = 11
)

type Node interface {
	GetNodeName() string
	GetNodeType() uint
}
