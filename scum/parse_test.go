package scum

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse_PlainText_RootHasOneTextChild(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := "hello world"

	ast := Parse(inp, &d, w)

	require.Len(t, ast.Nodes, 2, "root + one text node expected")
	require.Equal(t, NodeRoot, ast.Nodes[0].Type)
	require.Equal(t, NewSpan(0, len(inp)), ast.Nodes[0].Span)

	// root children
	require.Equal(t, 0, ast.Nodes[0].Children.Start)
	require.Equal(t, 1, ast.Nodes[0].Children.Len)
	require.Len(t, ast.ChildrenIdx, 1)
	require.Equal(t, 1, ast.ChildrenIdx[0])

	// child is text
	child := ast.Nodes[1]
	require.Equal(t, NodeText, child.Type)
	require.Equal(t, NewSpan(0, len(inp)), child.Span)
	require.Equal(t, inp, inp[child.Span.Start:child.Span.End])

	require.Empty(t, w.List())
}

func TestParse_GreedyCodeTag_CreatesTagNodeWithTextPayloadChild(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := "hi `code` bye"

	ast := Parse(inp, &d, w)

	// Expect: root + "hi " text + CODE tag node + " bye" text
	require.GreaterOrEqual(t, len(ast.Nodes), 4)

	root := ast.Nodes[0]
	require.Equal(t, NodeRoot, root.Type)

	// root should have 3 children in order: text, tag, text
	require.Equal(t, 3, root.Children.Len)
	require.Len(t, ast.ChildrenIdx, 3)

	firstChild := ast.Nodes[ast.ChildrenIdx[0]]
	require.Equal(t, NodeText, firstChild.Type)
	require.Equal(t, "hi ", inp[firstChild.Span.Start:firstChild.Span.End])

	tagNodeIdx := ast.ChildrenIdx[1]
	tagNode := ast.Nodes[tagNodeIdx]
	require.Equal(t, NodeTag, tagNode.Type)
	require.Equal(t, byte('`'), tagNode.TagID) // CODE trigger in testDict
	require.Equal(t, "`code`", inp[tagNode.Span.Start:tagNode.Span.End])

	// Tag node must have exactly one text child: payload "code"
	require.Equal(t, 1, tagNode.Children.Len)
	payloadIdx := ast.ChildrenIdx[tagNode.Children.Start]
	payloadNode := ast.Nodes[payloadIdx]
	require.Equal(t, NodeText, payloadNode.Type)
	require.Equal(t, "code", inp[payloadNode.Span.Start:payloadNode.Span.End])

	lastChild := ast.Nodes[ast.ChildrenIdx[2]]
	require.Equal(t, NodeText, lastChild.Type)
	require.Equal(t, " bye", inp[lastChild.Span.Start:lastChild.Span.End])

	require.Empty(t, w.List())
}

func TestParse_GreedyTag_WithAttribute_AttachesToTagNode(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := "`code`!lang{go}"

	ast := Parse(inp, &d, w)

	// root + tag + payload text at least
	require.GreaterOrEqual(t, len(ast.Nodes), 3)

	root := ast.Nodes[0]
	require.Equal(t, 1, root.Children.Len)
	tagNodeIdx := ast.ChildrenIdx[root.Children.Start]
	tagNode := ast.Nodes[tagNodeIdx]

	require.Equal(t, NodeTag, tagNode.Type)
	require.Equal(t, byte('`'), tagNode.TagID)
	require.Equal(t, "`code`", inp[tagNode.Span.Start:tagNode.Span.End])

	// Attribute must be on the tag node (per spec)
	require.Equal(t, 1, tagNode.Attributes.Len)
	require.Len(t, ast.Attributes, 1)

	attr := ast.Attributes[tagNode.Attributes.Start]
	require.Equal(t, "lang", inp[attr.Name.Start:attr.Name.End])
	require.Equal(t, "go", inp[attr.Payload.Start:attr.Payload.End])
	require.False(t, attr.IsFlag)

	require.Empty(t, w.List())
}
