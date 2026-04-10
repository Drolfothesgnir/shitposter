package sml

import (
	"fmt"
	"html"
	"strings"

	"github.com/Drolfothesgnir/shitposter/scum"
)

func handleTextNode(b *strings.Builder, _ *[]string, n scum.SerializableNode) {
	b.WriteString(html.EscapeString(n.Content))
}

func handleTagNode(b *strings.Builder, w *[]string, n scum.SerializableNode) {
	switch n.Name {
	case Bold:
		handleTag(b, w, n, attrMap{}, "strong", "strong")
	case Italic:
		handleTag(b, w, n, attrMap{}, "em", "em")
	case Underline:
		handleTag(b, w, n, attrMap{}, "span class=\"sml-underline\"", "span")
	case Link:
		handleTag(b, w, n, attrMap{"href": attrHref, "target": attrTarget, "title": attrTitle}, "a", "a")
	}
}

func handleNode(b *strings.Builder, w *[]string, n scum.SerializableNode) {
	if n.Type == "Tag" {
		handleTagNode(b, w, n)
		return
	}

	if n.Type == "Text" {
		handleTextNode(b, w, n)
		return
	}

	// Shouldn't happen
	*w = append(*w, fmt.Sprintf("SML: Unknown node type encountered: %s", n.Type))
}

func handleTag(b *strings.Builder, w *[]string, n scum.SerializableNode, m attrMap, start, end string) {
	b.WriteByte('<')
	b.WriteString(start)
	handleAttributes(b, w, m, n)
	b.WriteByte('>')
	for _, c := range n.Children {
		handleNode(b, w, c)
	}
	b.WriteString("</")
	b.WriteString(end)
	b.WriteByte('>')
}
