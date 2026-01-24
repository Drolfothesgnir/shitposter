package scum

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	input := "Hello, $$world\\!$$"
	out := Tokenize(&d, input, warns)

	require.Len(t, out.Tokens, 4)

	tree := Parse(input, &d, warns)
	require.Empty(t, warns.List())
	require.NotNil(t, tree)
	require.Len(t, tree.Nodes, 4)

	nodes := []Node{
		{
			Type:        NodeRoot,
			Span:        NewSpan(0, len(input)),
			FirstChild:  1,
			LastChild:   2,
			NextSibling: -1,
		},
		{
			Type:        NodeText,
			Span:        NewSpan(0, 7),
			NextSibling: 2,
			FirstChild:  -1,
			LastChild:   -1,
		},
		{
			Type:        NodeTag,
			TagID:       '$',
			Span:        NewSpan(7, 11),
			FirstChild:  3,
			LastChild:   3,
			NextSibling: -1,
		},
		{
			Type:        NodeText,
			Span:        NewSpan(9, 7),
			FirstChild:  -1,
			LastChild:   -1,
			NextSibling: -1,
		},
	}
	require.Equal(t, nodes[0], tree.Nodes[0])
	require.Equal(t, nodes[1], tree.Nodes[1])
	require.Equal(t, nodes[2], tree.Nodes[2])
	require.Equal(t, nodes[3], tree.Nodes[3])
}

func TestParse_PlainTextOnly(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	input := "just plain text"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())
	require.Len(t, tree.Nodes, 2) // root + text

	require.Equal(t, NodeRoot, tree.Nodes[0].Type)
	require.Equal(t, NodeText, tree.Nodes[1].Type)
	require.Equal(t, NewSpan(0, len(input)), tree.Nodes[1].Span)
}

func TestParse_EmptyInput(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	input := ""
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())
	require.Len(t, tree.Nodes, 1) // only root
	require.Equal(t, NodeRoot, tree.Nodes[0].Type)
}

func TestParse_NestedUniversalTags(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// $$*nested*$$
	input := "$$*nested*$$"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())
	// root -> BOLD -> ITALIC -> text
	require.Len(t, tree.Nodes, 4)

	// Root node
	require.Equal(t, NodeRoot, tree.Nodes[0].Type)
	require.Equal(t, 1, tree.Nodes[0].FirstChild)

	// BOLD tag
	require.Equal(t, NodeTag, tree.Nodes[1].Type)
	require.Equal(t, byte('$'), tree.Nodes[1].TagID)
	require.Equal(t, 2, tree.Nodes[1].FirstChild)

	// ITALIC tag
	require.Equal(t, NodeTag, tree.Nodes[2].Type)
	require.Equal(t, byte('*'), tree.Nodes[2].TagID)
	require.Equal(t, 3, tree.Nodes[2].FirstChild)

	// Text inside
	require.Equal(t, NodeText, tree.Nodes[3].Type)
}

func TestParse_OpeningClosingTagPair(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// [link text]
	input := "[link text]"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())
	// root -> LINK_TEXT_START -> text
	require.Len(t, tree.Nodes, 3)

	require.Equal(t, NodeRoot, tree.Nodes[0].Type)

	// LINK_TEXT_START tag with ID '['
	require.Equal(t, NodeTag, tree.Nodes[1].Type)
	require.Equal(t, byte('['), tree.Nodes[1].TagID)
	require.Equal(t, NewSpan(0, len(input)), tree.Nodes[1].Span)

	// Text inside the link
	require.Equal(t, NodeText, tree.Nodes[2].Type)
	require.Equal(t, "link text", input[tree.Nodes[2].Span.Start:tree.Nodes[2].Span.End])
}

func TestParse_GreedyTag(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// `code content`
	input := "`code content`"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())
	// root -> CODE tag -> text payload
	require.Len(t, tree.Nodes, 3)

	require.Equal(t, NodeRoot, tree.Nodes[0].Type)

	// CODE tag
	require.Equal(t, NodeTag, tree.Nodes[1].Type)
	require.Equal(t, byte('`'), tree.Nodes[1].TagID)

	// Payload text inside CODE
	require.Equal(t, NodeText, tree.Nodes[2].Type)
	require.Equal(t, "code content", input[tree.Nodes[2].Span.Start:tree.Nodes[2].Span.End])
}

func TestParse_AttributeKeyValue(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// [link]!URL{https://example.com}
	input := "[link]!URL{https://example.com}"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())
	// root -> LINK tag -> text
	require.Len(t, tree.Nodes, 3)

	// The LINK tag should have an attribute
	linkNode := tree.Nodes[1]
	require.Equal(t, 1, linkNode.Attributes.Len)

	attr := tree.Attributes[linkNode.Attributes.Start]
	require.False(t, attr.IsFlag)
	require.Equal(t, "URL", input[attr.Name.Start:attr.Name.End])
	require.Equal(t, "https://example.com", input[attr.Payload.Start:attr.Payload.End])
}

func TestParse_AttributeFlag(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// [link]!{nofollow}
	input := "[link]!{nofollow}"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())

	linkNode := tree.Nodes[1]
	require.Equal(t, 1, linkNode.Attributes.Len)

	attr := tree.Attributes[linkNode.Attributes.Start]
	require.True(t, attr.IsFlag)
	// For flag attributes, the name is stored in Payload (AttrKey is empty)
	require.Equal(t, "nofollow", input[attr.Payload.Start:attr.Payload.End])
}

func TestParse_MultipleAttributes(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// [link]!URL{https://x.com}!{nofollow}
	input := "[link]!URL{https://x.com}!{nofollow}"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())

	linkNode := tree.Nodes[1]
	require.Equal(t, 2, linkNode.Attributes.Len)

	attr1 := tree.Attributes[linkNode.Attributes.Start]
	require.False(t, attr1.IsFlag)
	require.Equal(t, "URL", input[attr1.Name.Start:attr1.Name.End])

	attr2 := tree.Attributes[linkNode.Attributes.Start+1]
	require.True(t, attr2.IsFlag)
	// For flag attributes, the name is in Payload
	require.Equal(t, "nofollow", input[attr2.Payload.Start:attr2.Payload.End])
}

func TestParse_DuplicateNestedTag_Warning(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// To trigger duplicate nested tag warning, we need a tag that's open but NOT at stack top.
	// [$$text$$] - BOLD inside LINK, then try to nest another BOLD inside
	// Actually, for universal tags, if at stack top, they close. So we need non-universal.
	// Use [outer [inner] outer] - nested LINK_TEXT_START
	input := "[outer [inner] outer]"
	tree := Parse(input, &d, warns)

	// Should have warning about duplicate nested tag (nested '[' while '[' is already open)
	require.NotEmpty(t, warns.List())
	require.Equal(t, IssueDuplicateNestedTag, warns.List()[0].Issue)

	// AST should still be valid
	require.NotNil(t, tree.Nodes)
	require.Equal(t, NodeRoot, tree.Nodes[0].Type)
}

func TestParse_MisplacedClosingTag_Warning(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// ] at the start without opening [
	input := "]some text"
	tree := Parse(input, &d, warns)

	require.NotEmpty(t, warns.List())
	require.Equal(t, IssueMisplacedClosingTag, warns.List()[0].Issue)

	// The ] should be treated as text, so we get text nodes
	require.True(t, len(tree.Nodes) >= 2)
}

func TestParse_MixedContent(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// Text $$bold$$ more *italic* end
	input := "Text $$bold$$ more *italic* end"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())

	// root -> text -> BOLD -> text(bold) -> text(more) -> ITALIC -> text(italic) -> text(end)
	// Count nodes: root + text + BOLD + text + text + ITALIC + text + text = 8
	require.Len(t, tree.Nodes, 8)

	// Check root has multiple children via sibling chain
	require.Equal(t, NodeRoot, tree.Nodes[0].Type)
	require.Equal(t, 1, tree.Nodes[0].FirstChild)
}

func TestParse_InfraWordRule(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// With RuleInfraWord, underscores in "file_name" are treated as plain text
	input := "file_name_here"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())
	// Should be just text, no UNDERLINE tags
	require.Len(t, tree.Nodes, 2) // root + text
	require.Equal(t, NodeText, tree.Nodes[1].Type)
}

func TestParse_UnderlineTagActivates(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// _underlined_ - underscores at boundaries should work
	input := "_underlined_"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())
	// root -> UNDERLINE tag -> text
	require.Len(t, tree.Nodes, 3)
	require.Equal(t, NodeTag, tree.Nodes[1].Type)
	require.Equal(t, byte('_'), tree.Nodes[1].TagID)
}

func TestParse_TagVsContentRule(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// ```code with ` backtick```
	input := "```code with ` backtick```"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())
	// root -> CODE tag -> text payload
	require.Len(t, tree.Nodes, 3)

	require.Equal(t, NodeTag, tree.Nodes[1].Type)
	require.Equal(t, byte('`'), tree.Nodes[1].TagID)

	// Payload should contain the content with single backtick
	payload := tree.Nodes[2]
	content := input[payload.Span.Start:payload.Span.End]
	require.Equal(t, "code with ` backtick", content)
}

func TestParse_ImageTag(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// :[alt text]
	input := ":[alt text]"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())
	// root -> IMAGE tag -> text
	require.Len(t, tree.Nodes, 3)

	require.Equal(t, NodeTag, tree.Nodes[1].Type)
	require.Equal(t, byte(':'), tree.Nodes[1].TagID)
}

func TestParse_EscapedTagNotTriggered(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// \$$ should not trigger BOLD tag
	input := "\\$$text"
	tree := Parse(input, &d, warns)

	// The escaped $ becomes text, then $text is text
	// No BOLD tag should be opened
	textFound := false
	for _, n := range tree.Nodes {
		if n.Type == NodeTag && n.TagID == '$' {
			t.Error("BOLD tag should not be triggered when escaped")
		}
		if n.Type == NodeText {
			textFound = true
		}
	}
	require.True(t, textFound)
}

func TestParse_ComplexNesting(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// [$$bold link$$]!URL{http://x.com}
	input := "[$$bold link$$]!URL{http://x.com}"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())

	// root -> LINK -> BOLD -> text
	// LINK should have URL attribute
	require.True(t, len(tree.Nodes) >= 4)

	linkNode := tree.Nodes[1]
	require.Equal(t, byte('['), linkNode.TagID)
	require.Equal(t, 1, linkNode.Attributes.Len)
}

func TestParse_AttributeOnText(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// Attributes attached to text node: "hello!STYLE{color: red}"
	input := "hello!STYLE{color: red}"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())

	// root -> text(hello) with attribute
	textNode := tree.Nodes[1]
	require.Equal(t, NodeText, textNode.Type)
	require.Equal(t, 1, textNode.Attributes.Len)

	attr := tree.Attributes[textNode.Attributes.Start]
	require.Equal(t, "STYLE", input[attr.Name.Start:attr.Name.End])
	require.Equal(t, "color: red", input[attr.Payload.Start:attr.Payload.End])
}

func TestParse_NestedTagSpan_ParentIncludesChildClosing(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// [*hi*] - nested tags, parent span should include child's closing tag
	// Positions: [ at 0, * at 1, h at 2, i at 3, * at 4, ] at 5
	input := "[*hi*]"
	tree := Parse(input, &d, warns)

	require.Empty(t, warns.List())
	// root -> '[' tag -> '*' tag -> text
	require.Len(t, tree.Nodes, 4)

	// The '[' tag (node 1) should span the entire "[*hi*]" from 0 to 6
	linkNode := tree.Nodes[1]
	require.Equal(t, NodeTag, linkNode.Type)
	require.Equal(t, byte('['), linkNode.TagID)
	require.Equal(t, 0, linkNode.Span.Start)
	require.Equal(t, len(input), linkNode.Span.End, "parent span should include child's closing tag")

	// The '*' tag (node 2) should span "*hi*" from 1 to 5
	italicNode := tree.Nodes[2]
	require.Equal(t, NodeTag, italicNode.Type)
	require.Equal(t, byte('*'), italicNode.TagID)
	require.Equal(t, 1, italicNode.Span.Start)
	require.Equal(t, 5, italicNode.Span.End)

	// Text "hi" (node 3) should span from 2 to 4
	textNode := tree.Nodes[3]
	require.Equal(t, NodeText, textNode.Type)
	require.Equal(t, 2, textNode.Span.Start)
	require.Equal(t, 4, textNode.Span.End)
}
