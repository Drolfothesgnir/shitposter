package markdown

// ImageNode represends a node with image URL and possible caption.
//
// It embeds URLNode to manage URL and caption text as BaseNode's children.
type ImageNode struct {
	*URLNode
}

// Markdown returns inner node's children markdown enclosed in image tag.
//
// Example: caption = "**beautiful** image of a *cat*", url = https://cat-image.com ->
// "![**beautiful** image of a *cat*](https://cat-image.com)".
func (n *ImageNode) Markdown() string {
	return markdownImage(n.BaseNode.Markdown(), n.URL)
}

// NewImageNode returns new *ImageNode with provided URL.
func NewImageNode(url string) Node {
	return &ImageNode{
		URLNode: NewURLNode(url).(*URLNode),
	}
}

// markdownImage returns URL markdown preppended with IMAGE marker ("!").
func markdownImage(caption, url string) string {
	return string(TagImageMarker) + markdownURL(caption, url)
}
