package markup

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActLinkTextStart_HappyPath_NotLastChar(t *testing.T) {
	input := "[a"
	i := 0

	var warns []Warning
	tok, stride := actLinkTextStart(input, i, &warns)

	require.Equal(t, 1, stride)
	require.Empty(t, warns)

	require.Equal(t, TypeLinkTextStart, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, "[", tok.Val)
}

func TestActLinkTextStart_LastChar_ReturnsTextAndWarning(t *testing.T) {
	input := "["
	i := 0

	var warns []Warning
	tok, stride := actLinkTextStart(input, i, &warns)

	require.Equal(t, 1, stride)

	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, "[", tok.Val)

	require.Len(t, warns, 1)
	w := warns[0]
	require.Equal(t, NodeText, w.Node)
	require.Equal(t, IssueUnexpectedEOL, w.Issue)
	require.Equal(t, 1, w.Index) // i + 1 per implementation
	require.Contains(t, w.Description, "Unexpected end of the line")
	require.Contains(t, w.Description, "[")
}

func TestActLinkTextStart_NotAtZeroIndex_Works(t *testing.T) {
	input := "x[y"
	i := 1

	var warns []Warning
	tok, stride := actLinkTextStart(input, i, &warns)

	require.Equal(t, 1, stride)
	require.Empty(t, warns)

	require.Equal(t, TypeLinkTextStart, tok.Type)
	require.Equal(t, 1, tok.Pos)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, "[", tok.Val)
}
