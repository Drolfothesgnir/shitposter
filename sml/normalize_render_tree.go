package sml

import (
	"fmt"

	"github.com/Drolfothesgnir/shitposter/scum"
)

// normalizeRenderTree strips the tree from reduntant and incorrect attributes.
func normalizeRenderTree(tree *scum.SerializableNode, issues *Issues) {
	for i := range tree.Children {
		c := &tree.Children[i]
		normalizeNode(c, issues)
	}
}

func normalizeNode(n *scum.SerializableNode, issues *Issues) {
	switch n.Type {
	case "Text":
		normalizeText(n, issues)
	case "Tag":
		normalizeTagNode(n, issues)
	default:
		panic(fmt.Sprintf("AAAAAAAAAAAAAA: data corrupted; unknown node type %q in the SML parsed tree.", n.Type))
	}
}

func normalizeTagNode(n *scum.SerializableNode, issues *Issues) {
	switch n.Name {
	case Bold, Italic, Underline:
		normalizeSimpleTag(n, issues)
	case Link:
		normalizeLink(n, issues)
	default:
		panic(fmt.Sprintf("AAAAAAAAAAAAAA: data corrupted; unknown tag %q in the SML parsed tree.", n.Name))
	}
}

func normalizeSimpleTag(n *scum.SerializableNode, issues *Issues) {
	// [Bold], [Italic] and [Underline] tags should have no attributes
	if len(n.Attributes) > 0 {
		// otherwise add an issue for each attribute
		for _, a := range n.Attributes {
			issues.Add(NewSyntaxIssueDescriptor(
				IssueAttributeNotAllowed,
				fmt.Sprintf("unknown attribute %q for the tag %q", a.Name, n.Name),
			))
		}
		// clear the attribute slice
		n.Attributes = n.Attributes[:0]
	}

	for i := range n.Children {
		c := &n.Children[i]
		normalizeNode(c, issues)
	}
}

func normalizeText(n *scum.SerializableNode, issues *Issues) {
	// text should not have any attributes
	if len(n.Attributes) > 0 {
		// otherwise add an issue for each attribute
		for _, a := range n.Attributes {
			issues.Add(NewSyntaxIssueDescriptor(
				IssueAttributeNotAllowed,
				fmt.Sprintf("unknown attribute %q for the text node", a.Name),
			))
		}

		// clear the attribute slice
		n.Attributes = n.Attributes[:0]
	}
}
