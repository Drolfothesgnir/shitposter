package scum

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSerialize_EmptyRoot(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	input := ""
	tree := Parse(input, &d, warns)

	result := tree.Serialize()

	require.Equal(t, "Root", result.Type)
	require.Equal(t, "ROOT", result.Name)
	require.Empty(t, result.Children)
	require.Equal(t, "", result.Content)
}

func TestSerialize_PlainText(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	input := "hello world"
	tree := Parse(input, &d, warns)

	result := tree.Serialize()

	require.Equal(t, "Root", result.Type)
	require.Len(t, result.Children, 1)

	// Text child
	textNode := result.Children[0]
	require.Equal(t, "Text", textNode.Type)
	require.Equal(t, "hello world", textNode.Content)
	require.Empty(t, textNode.Children)
}

func TestSerialize_SingleTag(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	input := "*italic*"
	tree := Parse(input, &d, warns)

	result := tree.Serialize()

	require.Len(t, result.Children, 1)

	// ITALIC tag
	tagNode := result.Children[0]
	require.Equal(t, "Tag", tagNode.Type)
	require.Equal(t, byte('*'), tagNode.ID)
	require.Equal(t, "*italic*", tagNode.Content)

	// Text inside tag
	require.Len(t, tagNode.Children, 1)
	require.Equal(t, "Text", tagNode.Children[0].Type)
	require.Equal(t, "italic", tagNode.Children[0].Content)
}

func TestSerialize_NestedTags(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	input := "[*nested*]"
	tree := Parse(input, &d, warns)

	result := tree.Serialize()

	require.Len(t, result.Children, 1)

	// LINK tag
	linkNode := result.Children[0]
	require.Equal(t, "Tag", linkNode.Type)
	require.Equal(t, byte('['), linkNode.ID)

	// ITALIC inside LINK
	require.Len(t, linkNode.Children, 1)
	italicNode := linkNode.Children[0]
	require.Equal(t, "Tag", italicNode.Type)
	require.Equal(t, byte('*'), italicNode.ID)

	// Text inside ITALIC
	require.Len(t, italicNode.Children, 1)
	require.Equal(t, "nested", italicNode.Children[0].Content)
}

func TestSerialize_MixedContent(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	input := "before *bold* after"
	tree := Parse(input, &d, warns)

	result := tree.Serialize()

	// root -> text, tag, text
	require.Len(t, result.Children, 3)

	require.Equal(t, "Text", result.Children[0].Type)
	require.Equal(t, "before ", result.Children[0].Content)

	require.Equal(t, "Tag", result.Children[1].Type)
	require.Equal(t, byte('*'), result.Children[1].ID)

	require.Equal(t, "Text", result.Children[2].Type)
	require.Equal(t, " after", result.Children[2].Content)
}

func TestSerialize_DifferentWidthTag(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// IMAGE tag: :[ opens (2), ] closes (1)
	input := ":[alt text]"
	tree := Parse(input, &d, warns)

	result := tree.Serialize()

	require.Len(t, result.Children, 1)

	imageNode := result.Children[0]
	require.Equal(t, "Tag", imageNode.Type)
	require.Equal(t, byte(':'), imageNode.ID)
	require.Equal(t, ":[alt text]", imageNode.Content)

	require.Len(t, imageNode.Children, 1)
	require.Equal(t, "alt text", imageNode.Children[0].Content)
}

func TestSerialize_DeeplyNested(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	// Use $$ and * which don't have special rules
	input := "[$$*deep*$$]"
	tree := Parse(input, &d, warns)

	result := tree.Serialize()

	// root -> [ -> $$ -> * -> text
	linkNode := result.Children[0]
	require.Equal(t, byte('['), linkNode.ID)

	boldNode := linkNode.Children[0]
	require.Equal(t, byte('$'), boldNode.ID)

	italicNode := boldNode.Children[0]
	require.Equal(t, byte('*'), italicNode.ID)

	textNode := italicNode.Children[0]
	require.Equal(t, "deep", textNode.Content)
}
