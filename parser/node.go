package parser

type TreeNode struct {
	data     int
	value    map[string]interface{}
	children []*TreeNode
}

func NewTreeNode(data int, value map[string]interface{}) *TreeNode {
	return &TreeNode{
		data:     data,
		value:    value,
		children: make([]*TreeNode, 0),
	}
}

func (node *TreeNode) AddChild(child *TreeNode) {
	node.children = append(node.children, child)
}

func (node *TreeNode) InsertChild(data int, value map[string]interface{}) *TreeNode {
	newChild := NewTreeNode(data, value)
	node.AddChild(newChild)
	return newChild
}
