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

	// NumNodeTypes is the total number of Node types. Should be placed as last const.
	NumNodeTypes
)

// Attribute represents a tag attribute.
// Name and Payload are spans into the original input string.
// IsFlag is true for flag attributes (no value).
type Attribute struct {
	Name    Span `json:"name"`
	Payload Span `json:"payload"`
	IsFlag  bool `json:"is_flag"`
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

	FirstChild  int
	LastChild   int
	NextSibling int

	// TODO: write tests for this
	// ChildCount is the number of children of this node.
	ChildCount int

	// Attributes defines the range in Attributes arena.
	Attributes Range
}

func NewNode() Node {
	return Node{
		Type:        NodeRoot,
		FirstChild:  -1,
		LastChild:   -1,
		NextSibling: -1,
		ChildCount:  0,
	}
}

// AST is the arena-backed abstract syntax tree.
type AST struct {
	// Input is the original source string.
	Input string

	// Nodes stores all AST nodes.
	Nodes []Node

	// Attributes stores all attributes referenced by nodes.
	Attributes []Attribute

	// MaxDepth is a meassurement of maximum embedding level of the AST.
	MaxDepth int

	// TextLength is the length of the text in the input.
	TextLength int

	// TotalTagNodes is the total count of the effective (no duplicate nesting) Tags in the input.
	TotalTagNodes int

	// TotalTextNodes is the total count of the plain text nodes in the input.
	TotalTextNodes int
}
