package dom

type NodeType uint

const (
	Node_Element               NodeType = 1
	Node_Attribute             NodeType = 2
	Node_Text                  NodeType = 3
	Node_CDATASection          NodeType = 4
	Node_ProcessingInstruction NodeType = 7
	Node_Comment               NodeType = 8
	Node_Document              NodeType = 9
	Node_DocumentType          NodeType = 10
	Node_DocumentFragment      NodeType = 11
)

type Node interface {
	GetNodeName() string
	GetNodeType() NodeType
}
