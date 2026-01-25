package scum

var nodeTypeToString = map[NodeType]string{
	NodeRoot: "Root",
	NodeTag:  "Tag",
	NodeText: "Text",
}

type SerializableAST struct {
	Tree       SerializableNode `json:"tree"`
	TextLength int              `json:"text_length"`
	Warnings   []Warning        `json:"warnings"`
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

// TODO: add Warnings to the result
// TODO: add text length to the result
func (ast AST) Serialize() (out SerializableAST) {
	root := ast.Nodes[0]

	tree := SerializableNode{
		Name:       "ROOT",
		Type:       nodeTypeToString[NodeRoot],
		Content:    ast.Input[root.Span.Start:root.Span.End],
		Children:   make([]SerializableNode, root.ChildCount),
		ID:         root.TagID,
		Attributes: ast.Attributes[root.Attributes.Start : root.Attributes.Start+root.Attributes.Len],
	}

	// seed stack with root's children
	stack := make([]serializeTask, 0, ast.MaxDepth)
	childIdx := root.FirstChild
	for i := 0; i < root.ChildCount; i++ {
		stack = append(stack, serializeTask{&tree, i, childIdx})
		childIdx = ast.Nodes[childIdx].NextSibling
	}

	for len(stack) > 0 {
		// pop
		task := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		node := ast.Nodes[task.nodeIdx]

		// build serializable node
		sn := SerializableNode{
			Type:       nodeTypeToString[node.Type],
			Content:    ast.Input[node.Span.Start:node.Span.End],
			Children:   make([]SerializableNode, node.ChildCount),
			ID:         node.TagID,
			Attributes: ast.Attributes[node.Attributes.Start : node.Attributes.Start+node.Attributes.Len],
		}

		// place in parent
		task.parent.Children[task.childIdx] = sn

		// push children tasks (need pointer to the placed node)
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

	out.Tree = tree
	out.TextLength = ast.TextLength
	out.Warnings = ast.Warnings.List()

	return out
}
