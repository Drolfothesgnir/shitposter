package scum

import "strings"

// Text returns all text node content concatenated in source order.
//
// The parser appends nodes while consuming tokens left-to-right, so a linear
// scan over the node arena preserves text order and avoids traversal overhead.
func (ast AST) Text() string {
	var b strings.Builder
	b.Grow(ast.TextByteLen)
	for _, n := range ast.Nodes {
		if n.Type != NodeText {
			continue
		}

		b.WriteString(ast.Input[n.Span.Start:n.Span.End])
	}
	return b.String()
}
