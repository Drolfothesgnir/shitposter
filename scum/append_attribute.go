package scum

func appendAttribute(ast *AST, parentIdx int, attr Attribute) {
	attrIdx := len(ast.Attributes)
	ast.Attributes = append(ast.Attributes, attr)

	parent := &ast.Nodes[parentIdx]

	if parent.Attributes.Len == 0 {
		parent.Attributes.Start = attrIdx
	}

	parent.Attributes.Len++
}
