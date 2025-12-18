package markdown

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestActCode_LastRune_IsTextWithWarning(t *testing.T) {
	substr := "`"
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, '`', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actCode(substr, cur, width, 0, true)

	require.True(t, ok)
	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, "`", tok.Val)
	require.Equal(t, 1, stride)

	require.Len(t, warns, 1)
	require.Equal(t, IssueUnexpectedEOL, warns[0].Issue)
	require.Equal(t, 1, warns[0].Index)
}

func TestActCode_UnclosedInline_ReturnsOpeningAsText(t *testing.T) {
	substr := "`code" // no closing `
	cur, width := utf8.DecodeRuneInString(substr)
	tok, warns, stride, ok := actCode(substr, cur, width, 0, false)

	require.True(t, ok)
	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 1, tok.Len)   // only opening tag is returned as text
	require.Equal(t, "`", tok.Val) // only opening backtick
	require.Equal(t, 1, stride)    // consumes only the opening tag, leaves the rest for next tokens

	require.Len(t, warns, 1)
	require.Equal(t, IssueUnclosedTag, warns[0].Issue)
	require.Equal(t, 0, warns[0].Index)
}

func TestActCode_UnclosedBlock_ReturnsWholeRestAsBlock(t *testing.T) {
	substr := "```code\nmore" // no closing ```
	cur, width := utf8.DecodeRuneInString(substr)
	tok, warns, stride, ok := actCode(substr, cur, width, 0, false)

	require.True(t, ok)
	require.Equal(t, TypeCodeBlock, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, len(substr), tok.Len)
	require.Equal(t, substr, tok.Val)
	require.Equal(t, len(substr), stride)

	require.Len(t, warns, 1)
	require.Equal(t, IssueUnclosedTag, warns[0].Issue)
	require.Equal(t, 0, warns[0].Index)
}

func TestActCode_ClosedInline_Simple(t *testing.T) {
	substr := "`code`"
	cur, width := utf8.DecodeRuneInString(substr)
	tok, warns, stride, ok := actCode(substr, cur, width, 0, false)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeCodeInline, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, len(substr), tok.Len)
	require.Equal(t, substr, tok.Val)
	require.Equal(t, len(substr), stride)
}

func TestActCode_Inline_NPlusOneRule_IgnoresShorterBacktickSequenceInside(t *testing.T) {
	// opening is "``" so a single "`" inside must NOT close it; closing is "``"
	substr := "``a`b``"
	cur, width := utf8.DecodeRuneInString(substr)
	tok, warns, stride, ok := actCode(substr, cur, width, 0, false)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeCodeInline, tok.Type) // openTagLen=2 (<3)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, len(substr), tok.Len)
	require.Equal(t, substr, tok.Val)
	require.Equal(t, len(substr), stride)
}

func TestActCode_Block_NPlusOneRule_IgnoresShorterBacktickSequenceInside(t *testing.T) {
	// opening is "```" so "``" inside must NOT close it; closing is "```"
	substr := "```a``b```"
	cur, width := utf8.DecodeRuneInString(substr)
	tok, warns, stride, ok := actCode(substr, cur, width, 0, false)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeCodeBlock, tok.Type) // openTagLen=3 (>=3)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, len(substr), tok.Len)
	require.Equal(t, substr, tok.Val)
	require.Equal(t, len(substr), stride)
}

func TestActCode_OnlyBackticks_UnclosedBlock(t *testing.T) {
	// all symbols are backticks -> openTagLen becomes full length, contentStartIdx==n, no closing tag found
	substr := "````"
	cur, width := utf8.DecodeRuneInString(substr)
	tok, warns, stride, ok := actCode(substr, cur, width, 0, false)

	require.True(t, ok)

	require.Equal(t, TypeCodeBlock, tok.Type) // openTagLen=4 (>=3)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, len(substr), tok.Len)
	require.Equal(t, substr, tok.Val)
	require.Equal(t, len(substr), stride)

	require.Len(t, warns, 1)
	require.Equal(t, IssueUnclosedTag, warns[0].Issue)
	require.Equal(t, 0, warns[0].Index)
}

// TODO: add alternative N+1 rule test with longer code sequence than tags
