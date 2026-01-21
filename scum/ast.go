package scum

// Range defines a slice in an arena-backed array.
// Start is the index in the arena, Len is the number of elements.
type Range struct {
	Start int
	Len   int
}

// NodeType defines the semantic kind of an AST node.
type NodeType int

const (
	NodeRoot NodeType = iota
	NodeText
	NodeTag
)

// Attribute represents a tag attribute.
// Name and Payload are spans into the original input string.
// IsFlag is true for flag attributes (no value).
type Attribute struct {
	Name    Span
	Payload Span
	IsFlag  bool
}

// Node represents a single AST node.
// Nodes are stored in an arena and linked via index ranges.
type Node struct {
	// Type indicates whether this node is text or a tag.
	Type NodeType

	// TagID is the tag identifier for NodeTag nodes.
	TagID byte

	// Span defines the byte range in Input covered by this node.
	Span Span

	// Children defines the range in ChildrenIdx arena.
	Children Range

	// Attributes defines the range in Attributes arena.
	Attributes Range
}

// AST is the arena-backed abstract syntax tree.
type AST struct {
	// Input is the original source string.
	Input string

	// Nodes stores all AST nodes.
	Nodes []Node

	// ChildrenIdx stores child node indices for all nodes.
	ChildrenIdx []int

	// Attributes stores all attributes referenced by nodes.
	Attributes []Attribute
}
