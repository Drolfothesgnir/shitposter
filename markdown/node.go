package markdown

// NodeType identifies the kind of node in the inline AST,
// e.g. "TEXT", "BOLD", "ITALIC", "LINK", etc.
type NodeType string

// Common Node types which can be used by different engines.
const (
	// NodeRoot represents the root of the AST.
	NodeRoot   NodeType = "ROOT"
	NodeBold   NodeType = "BOLD"
	NodeItalic NodeType = "ITALIC"
	NodeLink   NodeType = "LINK"
	NodeImage  NodeType = "IMAGE"
	NodeCode   NodeType = "CODE"
	NodeText   NodeType = "TEXT"
)

// Node represents a part of the inline AST. Some nodes are leaf nodes
// (e.g. TEXT), and some are containers that can carry child nodes
// (e.g. BOLD, ITALIC, LINK).
type Node interface {
	// NodeType returns the node's type, e.g. "BOLD", "ITALIC", "TEXT".
	NodeType() NodeType

	// Children returns this node's child nodes. Leaf nodes typically
	// return nil or an empty slice.
	Children() []Node

	// Append adds a new child node to this node.
	// Container nodes are expected to implement this;
	// leaf nodes may panic or ignore the call, depending on your design.
	Append(child Node)

	// DisplayText returns the "visible" text value of the node.
	//
	// For formatting nodes like BOLD and ITALIC it should return the
	// combined DisplayText of their children (i.e. what the user sees
	// without markdown syntax).
	//
	// For LINK nodes it should return the link text (children), if any.
	//
	// For IMAGE nodes it should return the image caption/alt text, if any.
	DisplayText() string

	// Value returns the semantic value of the node.
	//
	// For most nodes (TEXT, BOLD, ITALIC, etc.) it will typically be
	// the same as DisplayText().
	//
	// For LINK and IMAGE nodes it should return the actual URL.
	Value() string

	// Markdown returns a canonical markdown serialization of this node.
	//
	// For example:
	//   - BOLD node with child TEXT("hi") → "**hi**"
	//   - ITALIC node with child TEXT("hi") → "*hi*"
	//   - LINK node → "[text](url)"
	//   - IMAGE node → "![caption](url)"
	//   - TEXT node → its raw text value.
	Markdown() string

	// TextLength returns actual length of the text content, that is,
	// the count of letters in the plain text, not the count of bytes in the
	// raw content string.
	TextLength() int
}
