package markdown

import (
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestActEscape_LastRune_IsTextWithWarning(t *testing.T) {
	input := fmt.Sprintf("%11s", `\`)
	cur, width := utf8.DecodeLastRuneInString(input)
	require.Equal(t, '\\', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actEscape(input, 10)

	require.True(t, ok)
	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, 10, tok.Pos)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, `\`, tok.Val)

	require.Equal(t, 1, stride)

	require.Len(t, warns, 1)
	require.Equal(t, IssueRedundantEscape, warns[0].Issue)
	require.Equal(t, 10, warns[0].Index)
}

func TestActEscape_BeforeSpecialSymbol_NoWarning(t *testing.T) {
	substr := `\~`
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, '\\', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actEscape(substr, 0)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeEscapeSequence, tok.Type)
	require.Equal(t, 0, tok.Pos) // i
	require.Equal(t, 2, tok.Len) // "\" + "~"
	require.Equal(t, `\~`, tok.Val)

	require.Equal(t, 2, stride) // consumed "\" + "~"
}

func TestActEscape_BeforePlainText_WarnsRedundantEscape(t *testing.T) {
	input := fmt.Sprintf("% 7s", `\a`)
	cur, width := utf8.DecodeRuneInString(input[5:])
	require.Equal(t, '\\', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actEscape(input, 5)

	require.True(t, ok)

	require.Equal(t, TypeEscapeSequence, tok.Type)
	require.Equal(t, 5, tok.Pos)
	require.Equal(t, 2, tok.Len)
	require.Equal(t, `\a`, tok.Val)

	require.Equal(t, 2, stride)

	require.Len(t, warns, 1)
	require.Equal(t, IssueRedundantEscape, warns[0].Issue)
	require.Equal(t, 6, warns[0].Index) // nextIndex
	require.Equal(t, `\a`, warns[0].Near)
}

func TestActEscape_BeforeUTF8Rune_WarnsRedundantEscape(t *testing.T) {
	// "Ж" is 2 bytes in UTF-8.
	substr := "\\Ж"
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, '\\', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actEscape(substr, 0)

	require.True(t, ok)

	_, wNext := utf8.DecodeRuneInString(substr[width:])
	require.Equal(t, 2, wNext)

	require.Equal(t, TypeEscapeSequence, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 1+wNext, tok.Len) // "\" + "Ж"
	require.Equal(t, substr[:1+wNext], tok.Val)

	require.Equal(t, 1+wNext, stride)

	require.Len(t, warns, 1)
	require.Equal(t, IssueRedundantEscape, warns[0].Issue)
	require.Equal(t, 1, warns[0].Index)
	require.Equal(t, substr[:1+wNext], warns[0].Near)
}

func TestActEscape_BeforeEscape_NoWarning(t *testing.T) {
	substr := `\\`
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, '\\', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actEscape(substr, 0)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeEscapeSequence, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 2, tok.Len)
	require.Equal(t, `\\`, tok.Val)

	require.Equal(t, 2, stride) // consumed "\" + "\"
}
