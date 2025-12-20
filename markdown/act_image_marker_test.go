package markdown

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActImageMarker_HappyPath_ExclamationThenBracket(t *testing.T) {
	input := "![alt](url)"
	i := 0

	var warns []Warning
	tok, stride := actImageMarker(input, i, &warns)

	require.Equal(t, TypeImageTextStart, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 2, tok.Len)
	require.Equal(t, "![", tok.Val)

	require.Equal(t, 2, stride)
	require.Empty(t, warns)
}

func TestActImageMarker_LastChar_UnexpectedEOL(t *testing.T) {
	input := "!"
	i := 0

	var warns []Warning
	tok, stride := actImageMarker(input, i, &warns)

	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, "!", tok.Val)

	require.Equal(t, 1, stride)

	require.Len(t, warns, 1)
	w := warns[0]
	require.Equal(t, NodeText, w.Node)
	require.Equal(t, 0, w.Index)
	require.Equal(t, IssueUnexpectedEOL, w.Issue)
	require.Contains(t, w.Description, "expected to get")
	require.Equal(t, "", w.Near)
}

func TestActImageMarker_NextCharNotBracket_UnexpectedSymbol_ASCII(t *testing.T) {
	input := "!a"
	i := 0

	var warns []Warning
	tok, stride := actImageMarker(input, i, &warns)

	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, "!", tok.Val)

	require.Equal(t, 1, stride)

	require.Len(t, warns, 1)
	w := warns[0]
	require.Equal(t, NodeText, w.Node)
	require.Equal(t, 0, w.Index)
	require.Equal(t, IssueUnexpectedSymbol, w.Issue)
	require.Contains(t, w.Description, "expected to get")
	require.Contains(t, w.Description, "[")
	require.Contains(t, w.Description, "a")
	require.Equal(t, "!", w.Near) // current implementation sets near to current symbol only
}

func TestActImageMarker_NextCharNotBracket_UTF8(t *testing.T) {
	// next rune is multi-byte
	input := "!Ж"
	i := 0

	var warns []Warning
	tok, stride := actImageMarker(input, i, &warns)

	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, "!", tok.Val)

	require.Equal(t, 1, stride)

	require.Len(t, warns, 1)
	w := warns[0]
	require.Equal(t, IssueUnexpectedSymbol, w.Issue)
	require.Contains(t, w.Description, "Ж")
	require.Equal(t, "!", w.Near)
}

func TestActImageMarker_NotAtZeroIndex_StillWorks(t *testing.T) {
	input := "x![alt]"
	i := 1

	var warns []Warning
	tok, stride := actImageMarker(input, i, &warns)

	require.Equal(t, TypeImageTextStart, tok.Type)
	require.Equal(t, 1, tok.Pos)
	require.Equal(t, 2, tok.Len)
	require.Equal(t, "![", tok.Val)

	require.Equal(t, 2, stride)
	require.Empty(t, warns)
}
