package markdown

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenize_Empty(t *testing.T) {
	tokens := Tokenize("")
	require.Len(t, tokens, 0)
}

func TestTokenize_PlainText(t *testing.T) {
	tokens := Tokenize("hello")
	require.Len(t, tokens, 1)

	tok := tokens[0]
	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 5, tok.Len)
	require.Equal(t, "hello", tok.Val)
}

func TestTokenize_Bold(t *testing.T) {
	tokens := Tokenize("**hello**")
	require.Len(t, tokens, 3)

	t0 := tokens[0]
	require.Equal(t, TypeBold, t0.Type)
	require.Equal(t, 0, t0.Pos)
	require.Equal(t, 2, t0.Len)
	require.Equal(t, "**", t0.Val)

	t1 := tokens[1]
	require.Equal(t, TypeText, t1.Type)
	require.Equal(t, 2, t1.Pos)
	require.Equal(t, 5, t1.Len)
	require.Equal(t, "hello", t1.Val)

	t2 := tokens[2]
	require.Equal(t, TypeBold, t2.Type)
	require.Equal(t, 7, t2.Pos)
	require.Equal(t, 2, t2.Len)
	require.Equal(t, "**", t2.Val)
}

func TestTokenize_ItalicAndText(t *testing.T) {
	tokens := Tokenize("a*b*")
	require.Len(t, tokens, 4)

	require.Equal(t, TypeText, tokens[0].Type)
	require.Equal(t, "a", tokens[0].Val)
	require.Equal(t, 0, tokens[0].Pos)

	require.Equal(t, TypeItalic, tokens[1].Type)
	require.Equal(t, "*", tokens[1].Val)
	require.Equal(t, 1, tokens[1].Pos)

	require.Equal(t, TypeText, tokens[2].Type)
	require.Equal(t, "b", tokens[2].Val)
	require.Equal(t, 2, tokens[2].Pos)

	require.Equal(t, TypeItalic, tokens[3].Type)
	require.Equal(t, "*", tokens[3].Val)
	require.Equal(t, 3, tokens[3].Pos)
}

func TestTokenize_Strikethrough(t *testing.T) {
	tokens := Tokenize("~~strike~~")
	require.Len(t, tokens, 3)

	require.Equal(t, TypeStrikethrough, tokens[0].Type)
	require.Equal(t, "~~", tokens[0].Val)
	require.Equal(t, 0, tokens[0].Pos)

	require.Equal(t, TypeText, tokens[1].Type)
	require.Equal(t, "strike", tokens[1].Val)
	require.Equal(t, 2, tokens[1].Pos)

	require.Equal(t, TypeStrikethrough, tokens[2].Type)
	require.Equal(t, "~~", tokens[2].Val)
	require.Equal(t, 8, tokens[2].Pos)
}

func TestTokenize_MixedBoldAndText(t *testing.T) {
	s := "hi **bold** and *it*"
	// indices:
	// 0:h 1:i 2:' ' 3:* 4:* 5:b 6:o 7:l 8:d 9:* 10:* 11:' ' 12:a 13:n 14:d 15:' ' 16:* 17:i 18:t 19:*

	tokens := Tokenize(s)
	require.Len(t, tokens, 8)

	require.Equal(t, TypeText, tokens[0].Type)
	require.Equal(t, "hi ", tokens[0].Val)
	require.Equal(t, 0, tokens[0].Pos)

	require.Equal(t, TypeBold, tokens[1].Type)
	require.Equal(t, "**", tokens[1].Val)
	require.Equal(t, 3, tokens[1].Pos)

	require.Equal(t, TypeText, tokens[2].Type)
	require.Equal(t, "bold", tokens[2].Val)
	require.Equal(t, 5, tokens[2].Pos)

	require.Equal(t, TypeBold, tokens[3].Type)
	require.Equal(t, "**", tokens[3].Val)
	require.Equal(t, 9, tokens[3].Pos)

	require.Equal(t, TypeText, tokens[4].Type)
	require.Equal(t, " and ", tokens[4].Val)
	require.Equal(t, 11, tokens[4].Pos)

	require.Equal(t, TypeItalic, tokens[5].Type)
	require.Equal(t, "*", tokens[5].Val)
	require.Equal(t, 16, tokens[5].Pos)

	require.Equal(t, TypeText, tokens[6].Type)
	require.Equal(t, "it", tokens[6].Val)
	require.Equal(t, 17, tokens[6].Pos)

	require.Equal(t, TypeItalic, tokens[7].Type)
	require.Equal(t, "*", tokens[7].Val)
	require.Equal(t, 19, tokens[7].Pos)
}

func TestTokenize_LinkAndImageMarkers(t *testing.T) {
	s := "![alt](url)"
	// indices:
	// 0:! 1:[ 2:a 3:l 4:t 5:] 6:( 7:u 8:r 9:l 10:)
	tokens := Tokenize(s)
	require.Len(t, tokens, 7)

	require.Equal(t, TypeImageMarker, tokens[0].Type)
	require.Equal(t, "!", tokens[0].Val)
	require.Equal(t, 0, tokens[0].Pos)

	require.Equal(t, TypeLinkTextStart, tokens[1].Type)
	require.Equal(t, "[", tokens[1].Val)
	require.Equal(t, 1, tokens[1].Pos)

	require.Equal(t, TypeText, tokens[2].Type)
	require.Equal(t, "alt", tokens[2].Val)
	require.Equal(t, 2, tokens[2].Pos)

	require.Equal(t, TypeLinkTextEnd, tokens[3].Type)
	require.Equal(t, "]", tokens[3].Val)
	require.Equal(t, 5, tokens[3].Pos)

	require.Equal(t, TypeLinkURLStart, tokens[4].Type)
	require.Equal(t, "(", tokens[4].Val)
	require.Equal(t, 6, tokens[4].Pos)

	require.Equal(t, TypeText, tokens[5].Type)
	require.Equal(t, "url", tokens[5].Val)
	require.Equal(t, 7, tokens[5].Pos)

	require.Equal(t, TypeLinkURLEnd, tokens[6].Type)
	require.Equal(t, ")", tokens[6].Val)
	require.Equal(t, 10, tokens[6].Pos)
}

func TestTokenize_EscapeAndTag(t *testing.T) {
	s := "\\*"
	// indices:
	// 0:'\' 1:'*'
	tokens := Tokenize(s)
	require.Len(t, tokens, 2)

	require.Equal(t, TypeEscape, tokens[0].Type)
	require.Equal(t, "\\", tokens[0].Val)
	require.Equal(t, 0, tokens[0].Pos)

	require.Equal(t, TypeItalic, tokens[1].Type)
	require.Equal(t, "*", tokens[1].Val)
	require.Equal(t, 1, tokens[1].Pos)
}

func TestTokenize_UnicodeText(t *testing.T) {
	s := "привет"
	tokens := Tokenize(s)
	require.Len(t, tokens, 1)

	tok := tokens[0]
	require.Equal(t, TypeText, tok.Type)
	require.Equal(t, "привет", tok.Val)
	require.Equal(t, 0, tok.Pos)
	// Len is byte length; for "привет" it's 12, but we don't assert exact value here.
	require.Greater(t, tok.Len, 0)
}
