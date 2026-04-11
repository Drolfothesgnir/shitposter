package sml

import (
	"fmt"
	"html"
	"strings"

	"github.com/Drolfothesgnir/shitposter/scum"
)

func handleTextNode(b *strings.Builder, _ *Issues, n scum.SerializableNode) {
	b.WriteString(html.EscapeString(n.Content))
}

func handleTagNode(b *strings.Builder, i *Issues, n scum.SerializableNode) {
	switch n.Name {
	case Bold:
		handleTag(b, i, n, attrMap{}, "strong", "strong")
	case Italic:
		handleTag(b, i, n, attrMap{}, "em", "em")
	case Underline:
		handleTag(b, i, n, attrMap{}, "span class=\"sml-underline\"", "span")
	case Link:
		handleTag(b, i, n, attrMap{"href": attrHref, "target": attrTarget, "title": attrTitle}, "a", "a")
	}
}

func handleNode(b *strings.Builder, i *Issues, n scum.SerializableNode) {
	if n.Type == "Tag" {
		handleTagNode(b, i, n)
		return
	}

	if n.Type == "Text" {
		handleTextNode(b, i, n)
		return
	}

	// Shouldn't happen
	i.Add(NewSyntaxIssuesDescriptor(
		IssueUnknownNodeType,
		fmt.Sprintf("unknown node type encountered: %s", n.Type),
	))
}

func handleTag(b *strings.Builder, i *Issues, n scum.SerializableNode, m attrMap, start, end string) {
	b.WriteByte('<')
	b.WriteString(start)
	handleAttributes(b, i, m, n)
	b.WriteByte('>')
	for _, c := range n.Children {
		handleNode(b, i, c)
	}
	b.WriteString("</")
	b.WriteString(end)
	b.WriteByte('>')
}
