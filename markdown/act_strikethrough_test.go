package markdown

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestActStrikethrough_LastRune_IsTextWithWarning(t *testing.T) {
	substr := "~"
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, '~', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actStrikethrough(substr, cur, width, 7, true)

	require.True(t, ok)
	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, 7, tok.Pos)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, "~", tok.Val)

	require.Equal(t, 1, stride)

	require.Len(t, warns, 1)
	require.Equal(t, IssueUnexpectedEOL, warns[0].Issue)
	require.Equal(t, 7+1, warns[0].Index)
}

func TestActStrikethrough_SingleTilde_BeforeNonTilde_IsTextWithWarning(t *testing.T) {
	substr := "~a"
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, '~', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actStrikethrough(substr, cur, width, 0, false)

	require.True(t, ok)
	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, "~", tok.Val)

	require.Equal(t, 1, stride) // consume only the first '~'

	require.Len(t, warns, 1)
	require.Equal(t, IssueUnexpectedSymbol, warns[0].Issue)
	require.Equal(t, 1, warns[0].Index) // i + symLen
	require.Equal(t, "a", warns[0].Near)
}

func TestActStrikethrough_DoubleTilde_ProducesStrikethroughToken(t *testing.T) {
	substr := "~~"
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, '~', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actStrikethrough(substr, cur, width, 3, false)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeStrikethrough, tok.Type)
	require.Equal(t, 3, tok.Pos)
	require.Equal(t, 2, tok.Len)
	require.Equal(t, "~~", tok.Val)

	require.Equal(t, 2, stride) // consumed both '~'
}

func TestActStrikethrough_MoreThanTwoTildes_ConsumesOnlyFirstTwo(t *testing.T) {
	// actStrikethrough is only responsible for recognizing the tag "~~",
	// so on input "~~~" it should tokenize the first two and leave the last '~' for the outer loop.
	substr := "~~~"
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, '~', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actStrikethrough(substr, cur, width, 0, false)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeStrikethrough, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 2, tok.Len)
	require.Equal(t, "~~", tok.Val)

	require.Equal(t, 2, stride)
}
