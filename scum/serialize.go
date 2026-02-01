package scum

var mapNodeTypeToName [NumNodeTypes]string

func init() {
	mapNodeTypeToName[NodeRoot] = "Root"
	mapNodeTypeToName[NodeTag] = "Tag"
	mapNodeTypeToName[NodeText] = "Text"
}

type SerializableNode struct {
	Name       string             `json:"name"`
	Type       string             `json:"type"`
	Attributes []Attribute        `json:"attributes"`
	Content    string             `json:"content"`
	Children   []SerializableNode `json:"children"`
	ID         byte               `json:"id"`
}

type serializeTask struct {
	parent   *SerializableNode
	childIdx int // index in parent.Children
	nodeIdx  int // index in ast.Nodes
}

func (ast AST) Serialize() (tree SerializableNode) {
	root := ast.Nodes[0]

	// Single allocation for ALL child nodes (total nodes minus root)
	allChildren := make([]SerializableNode, len(ast.Nodes)-1)
	childrenUsed := 0

	tree = SerializableNode{
		Name:       "ROOT",
		Type:       mapNodeTypeToName[NodeRoot],
		Content:    ast.Input[root.Span.Start:root.Span.End],
		Children:   allChildren[childrenUsed : childrenUsed+root.ChildCount],
		ID:         root.TagID,
		Attributes: ast.Attributes[root.Attributes.Start : root.Attributes.Start+root.Attributes.Len],
	}
	childrenUsed += root.ChildCount

	stack := make([]serializeTask, 0, ast.MaxDepth)
	childIdx := root.FirstChild
	for i := 0; i < root.ChildCount; i++ {
		stack = append(stack, serializeTask{&tree, i, childIdx})
		childIdx = ast.Nodes[childIdx].NextSibling
	}

	for len(stack) > 0 {
		task := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		node := ast.Nodes[task.nodeIdx]

		// Slice from pre-allocated backing array
		children := allChildren[childrenUsed : childrenUsed+node.ChildCount]
		childrenUsed += node.ChildCount

		task.parent.Children[task.childIdx] = SerializableNode{
			Type:       mapNodeTypeToName[node.Type],
			Content:    ast.Input[node.Span.Start:node.Span.End],
			Children:   children,
			ID:         node.TagID,
			Attributes: ast.Attributes[node.Attributes.Start : node.Attributes.Start+node.Attributes.Len],
		}

		// push children tasks
		childIdx := node.FirstChild
		for i := 0; i < node.ChildCount; i++ {
			stack = append(stack, serializeTask{
				parent:   &task.parent.Children[task.childIdx],
				childIdx: i,
				nodeIdx:  childIdx,
			})
			childIdx = ast.Nodes[childIdx].NextSibling
		}
	}

	return
}
