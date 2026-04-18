package sml

import (
	"fmt"
	"html"
	"strings"

	"github.com/Drolfothesgnir/shitposter/scum"
)

// HTML returns the parsed input as an HTML.
func (p Poop) HTML() string {
	var b strings.Builder
	for _, n := range p.Tree.Children {
		renderNode(&b, n)
	}
	return b.String()
}

func renderTextNode(b *strings.Builder, n scum.SerializableNode) {
	b.WriteString(html.EscapeString(n.Content))
}

func renderTagNode(b *strings.Builder, n scum.SerializableNode) {
	switch n.Name {
	case Bold:
		renderTag(b, n, "strong", "strong")
	case Italic:
		renderTag(b, n, "em", "em")
	case Underline:
		renderTag(b, n, "span class=\"sml-internal-underline\"", "span")
	case Link:
		renderLink(b, n)
	default:
		panic(fmt.Sprintf("AAAAAAAAAAAA - data corruption; SML encountered unknown tag %q", n.Name))
	}
}

func renderNode(b *strings.Builder, n scum.SerializableNode) {
	if n.Type == "Tag" {
		renderTagNode(b, n)
		return
	}

	if n.Type == "Text" {
		renderTextNode(b, n)
		return
	}

	panic(fmt.Sprintf("AAAAAAAAAAAAA - data corruption; SML encountered unknown node type %q", n.Type))
}

func renderTag(b *strings.Builder, n scum.SerializableNode, start, end string) {
	b.WriteByte('<')
	b.WriteString(start)
	b.WriteByte('>')
	for _, c := range n.Children {
		renderNode(b, c)
	}
	b.WriteString("</")
	b.WriteString(end)
	b.WriteByte('>')
}

func renderLink(b *strings.Builder, n scum.SerializableNode) {
	b.WriteString("<a")
	for _, a := range n.Attributes {
		b.WriteByte(' ')
		b.WriteString(a.Name)
		b.WriteString(`="`)
		b.WriteString(html.EscapeString(a.Payload))
		b.WriteByte('"')
	}
	b.WriteByte('>')
	for _, c := range n.Children {
		renderNode(b, c)
	}
	b.WriteString("</a>")
}
