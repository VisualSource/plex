package dom

type CommentNode struct {
	data string
}

func (c *CommentNode) GetNodeType() NodeType {
	return Node_Comment
}

func (c *CommentNode) GetNodeName() string {
	return "#comment"
}

func NewCommentNode(data string) *CommentNode {
	return &CommentNode{
		data: data,
	}
}
