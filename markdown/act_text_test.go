package markdown

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestActText_ConsumesUntilFirstSpecialSymbol(t *testing.T) {
	// special symbols (per isSpecialSymbol): '\', '~', '`'
	substr := "hello~world"
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, 'h', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actText(substr, cur, width, 0, false)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, len("hello"), tok.Len)
	require.Equal(t, "hello", tok.Val)

	require.Equal(t, len("hello"), stride)
}

func TestActText_ConsumesEntireString_WhenNoSpecialSymbols(t *testing.T) {
	substr := "just plain text"
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, 'j', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actText(substr, cur, width, 5, false)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, 5, tok.Pos)
	require.Equal(t, len(substr), tok.Len)
	require.Equal(t, substr, tok.Val)

	require.Equal(t, len(substr), stride)
}

func TestActText_WhenFirstRuneIsSpecial_ReturnsEmptyTextToken(t *testing.T) {
	// This is a bit unusual, but it documents the current behavior:
	// actText will see the first rune as special and return an empty text token (Len=0, Val="").
	// In your Tokenize() implementation this shouldn't happen because you switch to actEscape/actCode/actStrikethrough,
	// but the test captures actText behavior in isolation.
	substr := "`code`"
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, '`', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actText(substr, cur, width, 0, false)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 0, tok.Len)
	require.Equal(t, "", tok.Val)

	require.Equal(t, 0, stride)
}

func TestActText_UTF8_ConsumesCorrectByteLength(t *testing.T) {
	// "Ж" is 2 bytes in UTF-8. Special symbol is backtick.
	substr := "ЖЖЖ`x"
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, 'Ж', cur)
	require.Equal(t, 2, width)

	tok, warns, stride, ok := actText(substr, cur, width, 0, false)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, 0, tok.Pos)

	// 3 * 2 bytes
	require.Equal(t, 6, tok.Len)
	require.Equal(t, "ЖЖЖ", tok.Val)
	require.Equal(t, 6, stride)
}

func TestActText_StopsBeforeEscapeSymbol(t *testing.T) {
	substr := "hi\\there"
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, 'h', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actText(substr, cur, width, 0, false)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, "hi", tok.Val)
	require.Equal(t, len("hi"), tok.Len)
	require.Equal(t, len("hi"), stride)
}
