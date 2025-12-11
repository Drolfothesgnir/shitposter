package markdown

// URLNode represents Node used for storing HTTP links.
//
// The address is stored in the URL field and all inner markdown text
// is managed by the embedded BaseNode.
type URLNode struct {
	URL string `json:"url"`
	*BaseNode
}

// DisplayText returns all the internal plain text, if provided.
//
// Example in the markdown "[google](https://google.com)" DisplayText will return "google".
func (n *URLNode) DisplayText() string {
	return n.BaseNode.DisplayText()
}

// Value returns URL of the link.
func (n *URLNode) Value() string {
	return n.URL
}

// Markdown returns aggregated child nodes' markdown enclosed into link text tags and the URL
// enclosed in the URL tags.
//
// Example: inner markdown = "**Yahoo\!**", url = "https://yahoo.com" -> "[**Yahoo\!**](https://yahoo.com)".
func (n *URLNode) Markdown() string {
	return markdownURL(n.BaseNode.Markdown(), n.URL)
}

// NewURLNode creates new *URLNode with a given URL.
func NewURLNode(url string) Node {
	return &URLNode{
		URL:      url,
		BaseNode: NewBaseNode(NodeLink),
	}
}

// markdownURL returns display text and url enclosed in URL tags.
//
// Example: markdown = "***some text***", url = "https://address.com" -> "[***some text***](https://address.com)".
func markdownURL(innerMarkdown, url string) string {
	return string(TagLinkTextStart) +
		innerMarkdown +
		string(TagLinkTextEnd) +
		string(TagLinkURLStart) +
		url +
		string(TagLinkURLEnd)
}
