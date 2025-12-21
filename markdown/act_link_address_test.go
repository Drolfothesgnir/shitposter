package markdown

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActLinkAddress_HappyPath_URLPresent(t *testing.T) {
	input := "abc](https://google.com)zzz"
	// trigger on the ']' (SymbolLinkTextEnd)
	i := 3

	var warns []Warning
	tok, stride := actLinkAddress(input, i, &warns)

	require.Empty(t, warns)

	require.Equal(t, TypeLinkAddress, tok.Type)
	require.Equal(t, i, tok.Pos)
	require.Equal(t, "](https://google.com)", tok.Val)
	require.Equal(t, len("](https://google.com)"), tok.Len)

	require.Equal(t, len("](https://google.com)"), stride)
}

func TestActLinkAddress_HappyPath_StopsAtFirstClosingParen(t *testing.T) {
	// Should close at the first ')', leaving the rest for the main loop.
	input := "x](a)b)tail"
	i := 1

	var warns []Warning
	tok, stride := actLinkAddress(input, i, &warns)

	require.Empty(t, warns)

	require.Equal(t, TypeLinkAddress, tok.Type)
	require.Equal(t, "](a)", tok.Val)
	require.Equal(t, len("](a)"), tok.Len)
	require.Equal(t, len("](a)"), stride)
}

func TestActLinkAddress_EmptyURL_ReturnsTypeLinkAddressAndWarning(t *testing.T) {
	input := "x]()tail"
	i := 1

	var warns []Warning
	tok, stride := actLinkAddress(input, i, &warns)

	require.Equal(t, TypeLinkAddress, tok.Type)
	require.Equal(t, "]()", tok.Val)
	require.Equal(t, 3, tok.Len)
	require.Equal(t, 3, stride)

	require.Len(t, warns, 1)
	w := warns[0]
	require.Equal(t, NodeLink, w.Node)
	require.Equal(t, IssueMalformedLink, w.Issue)
	require.Equal(t, i+1, w.Index)
	require.Equal(t, "]()", w.Near)
	require.Contains(t, w.Description, "Empty URL")
}

func TestActLinkAddress_UnclosedURL_ReturnsTextTwoCharsAndWarning(t *testing.T) {
	// Missing closing ')'
	input := "abc](https://google.com"
	i := 3

	var warns []Warning
	tok, stride := actLinkAddress(input, i, &warns)

	// Should return "](" as text
	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, i, tok.Pos)
	require.Equal(t, "](", tok.Val)
	require.Equal(t, 2, tok.Len)
	require.Equal(t, 2, stride)

	require.Len(t, warns, 1)
	w := warns[0]
	require.Equal(t, NodeText, w.Node)
	require.Equal(t, IssueUnexpectedEOL, w.Issue)
	require.Equal(t, len(input), w.Index) // implementation uses Index: n
	require.Contains(t, w.Description, "doesn't contain")
}

func TestActLinkAddress_RightAfterBracket_NotParen_UnexpectedSymbol_ASCII(t *testing.T) {
	input := "a]x"
	i := 1

	var warns []Warning
	tok, stride := actLinkAddress(input, i, &warns)

	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, "]", tok.Val)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, 1, stride)

	require.Len(t, warns, 1)
	w := warns[0]
	require.Equal(t, NodeText, w.Node)
	require.Equal(t, IssueUnexpectedSymbol, w.Issue)
	require.Equal(t, i+1, w.Index)
	require.Equal(t, "]x", w.Near)
	require.Contains(t, w.Description, "expected to find")
	require.Contains(t, w.Description, "(")
	require.Contains(t, w.Description, "x")
}

func TestActLinkAddress_RightAfterBracket_NotParen_UnexpectedSymbol_UTF8(t *testing.T) {
	// next rune after ']' is multi-byte 'Ж'
	input := "a]Ж"
	i := 1

	var warns []Warning
	tok, stride := actLinkAddress(input, i, &warns)

	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, "]", tok.Val)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, 1, stride)

	require.Len(t, warns, 1)
	w := warns[0]
	require.Equal(t, IssueUnexpectedSymbol, w.Issue)
	require.Equal(t, i+1, w.Index)

	// Near should include ']' plus the UTF-8 rune bytes of 'Ж'
	require.True(t, len(w.Near) > 1, "Near should include more than just ']'")
	require.Equal(t, byte(']'), w.Near[0])
	require.Contains(t, w.Description, "Ж")
}

func TestActLinkAddress_BracketIsLastChar_UnexpectedEOL(t *testing.T) {
	input := "abc]"
	i := 3

	var warns []Warning
	tok, stride := actLinkAddress(input, i, &warns)

	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, "]", tok.Val)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, 1, stride)

	require.Len(t, warns, 1)
	w := warns[0]
	require.Equal(t, NodeText, w.Node)
	require.Equal(t, IssueUnexpectedEOL, w.Issue)
	require.Equal(t, len(input), w.Index) // Index: n
	require.Contains(t, w.Description, "got EOL")
	require.Contains(t, w.Description, "(")
}

func TestActLinkAddress_NextIsParenButParenIsLast_Unclosed_ReturnsTextAndWarning(t *testing.T) {
	// i points to ']' and the next char is '(' but there is nothing else, so closing ')' not found.
	input := "]("
	i := 0

	var warns []Warning
	tok, stride := actLinkAddress(input, i, &warns)

	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, "](", tok.Val)
	require.Equal(t, 2, tok.Len)
	require.Equal(t, 2, stride)

	require.Len(t, warns, 1)
	w := warns[0]
	require.Equal(t, IssueUnexpectedEOL, w.Issue)
	require.Equal(t, len(input), w.Index)
}

func TestActLinkAddress_EmptyURL_AtEnd(t *testing.T) {
	// exactly "]()"
	input := "]()"
	i := 0

	var warns []Warning
	tok, stride := actLinkAddress(input, i, &warns)

	require.Equal(t, TypeLinkAddress, tok.Type)
	require.Equal(t, "]()", tok.Val)
	require.Equal(t, 3, tok.Len)
	require.Equal(t, 3, stride)

	require.Len(t, warns, 1)
	require.Equal(t, IssueMalformedLink, warns[0].Issue)
}

func TestActLinkAddress_ValidURL_WithSpacesInside_IsStillNonEmpty(t *testing.T) {
	// actLinkAddress doesn't validate URL syntax; it only checks non-empty content between '(' and ')'
	input := "x](   )y"
	i := 1

	var warns []Warning
	tok, stride := actLinkAddress(input, i, &warns)

	require.Empty(t, warns)

	require.Equal(t, TypeLinkAddress, tok.Type)
	require.Equal(t, "](   )", tok.Val)
	require.Equal(t, len("](   )"), tok.Len)
	require.Equal(t, len("](   )"), stride)
}
