package markup

import "strings"

// BaseNode represents basic element of the AST.
//
// It has core node logic and fields to be used in the AST.
type BaseNode struct {
	Type       NodeType `json:"type"`
	ChildNodes []Node   `json:"children,omitempty"`
}

// NodeType implements Node.NodeType method by returning BaseNode's type.
func (n *BaseNode) NodeType() NodeType {
	return n.Type
}

// Children implements Node.Children method by returning BaseNode's child nodes.
func (n *BaseNode) Children() []Node {
	return n.ChildNodes
}

// Append implements Node.Append method by attaching new Node to the BaseNode's child list.
func (n *BaseNode) Append(child Node) {
	n.ChildNodes = append(n.ChildNodes, child)
}

// DisplayText implements Node.DisplayText method by concatenating
// display texts of all of the BaseNode's children.
func (n *BaseNode) DisplayText() string {
	var b strings.Builder

	for _, c := range n.ChildNodes {
		b.WriteString(c.DisplayText())
	}

	return b.String()
}

// Value implements Node.Value method by concatenating
// values of all of the BaseNode's children.
func (n *BaseNode) Value() string {
	var b strings.Builder

	for _, c := range n.ChildNodes {
		b.WriteString(c.Value())
	}

	return b.String()
}

// Markdown implements Node.Markdown method by concatenating
// markdown strings of all of the BaseNode's children.
func (n *BaseNode) Markdown() string {
	var b strings.Builder

	for _, c := range n.ChildNodes {
		b.WriteString(c.Markdown())
	}

	return b.String()
}

// TextLength returns combined text length of the node's children.
func (n *BaseNode) TextLength() int {
	sum := 0
	for _, c := range n.ChildNodes {
		sum += c.TextLength()
	}
	return sum
}

// NewBaseNode is a factory method for creating
// a BaseNode with a given type.
func NewBaseNode(nodeType NodeType) *BaseNode {
	return &BaseNode{Type: nodeType}
}
