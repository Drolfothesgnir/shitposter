package scum

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
