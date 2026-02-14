package scum

// appendNode appends node to the [AST.Nodes] arena and links it as the last
// child of the parent at parentIdx. It maintains the parent's FirstChild,
// LastChild, NextSibling linked list and increments [Node.ChildCount].
// Returns the index of the newly appended node.
func appendNode(ast *AST, parentIdx int, node Node) int {
	nodeIdx := len(ast.Nodes)
	ast.Nodes = append(ast.Nodes, node)

	parent := &ast.Nodes[parentIdx]
	parent.ChildCount++

	if parent.FirstChild == -1 {
		parent.FirstChild = nodeIdx
		parent.LastChild = nodeIdx
		return nodeIdx
	}

	lastChild := &ast.Nodes[parent.LastChild]
	lastChild.NextSibling = nodeIdx
	parent.LastChild = nodeIdx

	return nodeIdx
}
