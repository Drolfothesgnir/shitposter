package sml

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/Drolfothesgnir/shitposter/scum"
)

func handleTextNode(b *strings.Builder, _ *[]string, n scum.SerializableNode) {
	b.WriteString(html.EscapeString(n.Content))
}

func handleTagNode(b *strings.Builder, w *[]string, n scum.SerializableNode) {
	switch n.Name {
	case Bold:
		handleTag(b, w, n, "strong", "strong")
	case Italic:
		handleTag(b, w, n, "em", "em")
	case Underline:
		handleTag(b, w, n, "span class=\"underline\"", "span")
	case Link:
		handleTag(b, w, n, "a", "a")
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

// TODO: check if attributes are appropriate
func handleAttributes(b *strings.Builder, _ *[]string, n scum.SerializableNode) {
	for _, a := range n.Attributes {
		// TODO: check if scum strips whitespaces
		var name, payload string
		if a.IsFlag {
			name = strings.ToLower(a.Payload)
			payload = "\"true\""
		} else {
			name = strings.ToLower(a.Name)
			payload = strconv.Quote(a.Payload)
		}

		b.WriteByte(' ')
		b.WriteString(name)
		b.WriteString("=")
		b.WriteString(payload)
	}
}

func handleTag(b *strings.Builder, w *[]string, n scum.SerializableNode, start, end string) {
	b.WriteByte('<')
	b.WriteString(start)
	handleAttributes(b, w, n)
	b.WriteByte('>')
	for _, c := range n.Children {
		handleNode(b, w, c)
	}
	b.WriteString("</")
	b.WriteString(end)
	b.WriteByte('>')
}
