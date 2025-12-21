package markup

import "unicode/utf8"

// TextNode represents plain text as a Node.
//
// It is forced to be a leaf in a tree, since it has Append method overriden to be a no-op.
type TextNode struct {
	// Content is a plain text stored in the node.
	Content string `json:"value"`
	*BaseNode
}

// DisplayText returns plain text stored in the node.
func (n *TextNode) DisplayText() string {
	return n.Content
}

// Value returns plain text stored in the node.
func (n *TextNode) Value() string {
	return n.Content
}

// Markdown returns plain text stored in the node.
func (n *TextNode) Markdown() string {
	return n.Content
}

// TextLength returns letter (not byte!) count of the plain text
// stored in the Content field.
func (n *TextNode) TextLength() int {
	return utf8.RuneCountInString(n.Content)
}

// Append method is no-op by design to explicitely forbid a TextNode
// from having children.
func (n *TextNode) Append(child Node) {
	// no-op
}

// NewTextNode creates new *TextNode with provided text content.
func NewTextNode(content string) *TextNode {
	return &TextNode{
		Content:  content,
		BaseNode: NewBaseNode(NodeText),
	}
}
