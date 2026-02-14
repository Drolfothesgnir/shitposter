package scum

// appendAttribute appends attr to the [AST.Attributes] arena and links it to
// the node at parentIdx. The first attribute sets [Node.Attributes].Start;
// subsequent attributes only increment Len (attributes for a single node are
// always contiguous in the arena).
func appendAttribute(ast *AST, parentIdx int, attr Attribute) {
	attrIdx := len(ast.Attributes)
	ast.Attributes = append(ast.Attributes, attr)

	parent := &ast.Nodes[parentIdx]

	if parent.Attributes.Len == 0 {
		parent.Attributes.Start = attrIdx
	}

	parent.Attributes.Len++
}
