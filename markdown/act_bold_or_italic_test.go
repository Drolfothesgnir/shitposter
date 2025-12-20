package markdown

import (
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestActBoldOrItalic_LastRune_ReturnsItalic(t *testing.T) {
	input := fmt.Sprintf("%12s", "*")
	cur, width := utf8.DecodeLastRuneInString(input)
	require.Equal(t, '*', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actBoldOrItalic(input, 11)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeItalic, tok.Type)
	require.Equal(t, 11, tok.Pos)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, "*", tok.Val)

	require.Equal(t, 1, stride)
}

func TestActBoldOrItalic_SingleAsteriskBeforeNonAsterisk_ReturnsItalic(t *testing.T) {
	substr := "*a"
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, '*', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actBoldOrItalic(substr, 0)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeItalic, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, "*", tok.Val)

	require.Equal(t, 1, stride)
}

func TestActBoldOrItalic_DoubleAsterisk_ReturnsBold(t *testing.T) {
	input := fmt.Sprintf("% 5s", "**")
	cur, width := utf8.DecodeRuneInString(input[3:])
	require.Equal(t, '*', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actBoldOrItalic(input, 3)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeBold, tok.Type)
	require.Equal(t, 3, tok.Pos)
	require.Equal(t, 2, tok.Len)
	require.Equal(t, "**", tok.Val)

	require.Equal(t, 2, stride)
}

func TestActBoldOrItalic_TripleAsterisk_ConsumesTwoAsBold(t *testing.T) {
	// actBoldOrItalic only decides between "*" and "**" at the current position.
	// For "***" it should produce a bold token for the first two asterisks,
	// leaving the last "*" for the outer loop.
	substr := "***"
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, '*', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actBoldOrItalic(substr, 0)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeBold, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 2, tok.Len)
	require.Equal(t, "**", tok.Val)

	require.Equal(t, 2, stride)
}

func TestActBoldOrItalic_UTF8AfterAsterisk_DoesNotAffectWidth(t *testing.T) {
	// Next rune is 'Ж' (2 bytes), but since it's not '*', token must be italic with Len=1.
	substr := "*Ж"
	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, '*', cur)
	require.Equal(t, 1, width)

	tok, warns, stride, ok := actBoldOrItalic(substr, 0)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeItalic, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, "*", tok.Val)

	require.Equal(t, 1, stride)
}
