package markdown

// FormatNode represents particular node for string formatting, e.g. type BOLD or ITALIC.
//
// FormatNode embeds BaseNode and ovverides its Markdown method.
type FormatNode struct {
	*BaseNode
}

// Markdown extracts concatenated markdown of its children
// and wraps it with the tags for it's type, e.g. '**' for BOLD type and '*'/'_' for ITALIC,
// if the node's type has corresponding formatter function inside the type-to-formatter map.
func (n *FormatNode) Markdown() string {
	formatter, ok := typeToFormatter[n.Type]

	// extracting markdown of the children
	inner := n.BaseNode.Markdown()

	// if the node has its corresponding formatter
	// return wrapped inner markdown
	if ok {
		return formatter(inner)
	}

	// return raw child markdown otherwise
	return inner
}

// NewFormatNode creates new *FormatNode with a given type.
func NewFormatNode(nodeType NodeType) *FormatNode {
	return &FormatNode{
		BaseNode: NewBaseNode(nodeType),
	}
}

// typeToFormatter is used to map NodeTypes of the formatter nodes (BOLD, ITALIC)
// to helper markdown wrapping functions.
var typeToFormatter = map[NodeType]func(s string) string{
	NodeBold:   markdownBold,
	NodeItalic: markdownItalic,
}

// mardownBold returns a provided string with attached BOLD tags ('**') on the sides.
//
// Example: "hello" -> "**hello**".
func markdownBold(s string) string {
	return string(TagBold) + s + string(TagBold)
}

// markdownItalic returns a provided string with attached ITALIC tags ('*') on the sides.
//
// Example: "hello" -> "*hello*".
func markdownItalic(s string) string {
	return string(TagItalic) + s + string(TagItalic)
}
