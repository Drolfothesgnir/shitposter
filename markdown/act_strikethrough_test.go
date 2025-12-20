package markdown

import (
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestActStrikethrough_LastRune_IsTextWithWarning(t *testing.T) {
	input := fmt.Sprintf("% 8s", "~")
	cur, width := utf8.DecodeLastRuneInString(input)
	require.Equal(t, '~', cur)
	require.Equal(t, 1, width)

	warns := make([]Warning, 0)

	tok, stride := actStrikethrough(input, 7, &warns)

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

	warns := make([]Warning, 0)

	tok, stride := actStrikethrough(substr, 0, &warns)

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
	input := fmt.Sprintf("% 5s", "~~")
	cur, width := utf8.DecodeRuneInString(input[3:])
	require.Equal(t, '~', cur)
	require.Equal(t, 1, width)

	warns := make([]Warning, 0)

	tok, stride := actStrikethrough(input, 3, &warns)

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

	warns := make([]Warning, 0)

	tok, stride := actStrikethrough(substr, 0, &warns)

	require.Empty(t, warns)

	require.Equal(t, TypeStrikethrough, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 2, tok.Len)
	require.Equal(t, "~~", tok.Val)

	require.Equal(t, 2, stride)
}
