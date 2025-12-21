package markup

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func prevRuneAt(input string, i int) rune {
	if i <= 0 {
		return rune(0)
	}
	r, _ := utf8.DecodeLastRuneInString(input[:i])
	return r
}

func TestIsUnderlineTag_Boundaries(t *testing.T) {
	cases := []struct {
		name  string
		input string
		i     int
		want  bool
	}{
		{
			name:  "start_of_string_is_tag",
			input: "_hello",
			i:     0,
			want:  true,
		},
		{
			name:  "end_of_string_is_tag",
			input: "hello_",
			i:     5,
			want:  true,
		},
		{
			name:  "between_space_and_alnum_is_tag",
			input: " _hello",
			i:     1,
			want:  true,
		},
		{
			name:  "between_alnum_and_space_is_tag",
			input: "hello_ world",
			i:     5,
			want:  true,
		},
		{
			name:  "between_two_alnums_is_not_tag",
			input: "hello_world",
			i:     5,
			want:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := len(tc.input)
			require.True(t, tc.i >= 0 && tc.i < n, "bad index for test case")
			require.Equal(t, byte('_'), tc.input[tc.i], "test index must point to '_'")

			prev := prevRuneAt(tc.input, tc.i)
			got := isUnderlineTag(tc.input, tc.i, n, prev)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestIsUnderlineTag_AdjacentUnderscores_AreNotTags(t *testing.T) {
	// "__hello": first '_' is not a tag because next is '_' (run suppression)
	input := "__hello"
	n := len(input)

	got0 := isUnderlineTag(input, 0, n, prevRuneAt(input, 0))
	require.False(t, got0)

	got1 := isUnderlineTag(input, 1, n, prevRuneAt(input, 1))
	require.False(t, got1) // previous rune is '_' => not a tag
}

func TestIsUnderlineTag_DoubleUnderscoreInMiddle_NotTags(t *testing.T) {
	// "hello__world": both underscores are suppressed as plain text
	input := "hello__world"
	n := len(input)

	// first underscore (next is underscore)
	gotFirst := isUnderlineTag(input, 5, n, prevRuneAt(input, 5))
	require.False(t, gotFirst)

	// second underscore (prev is underscore)
	gotSecond := isUnderlineTag(input, 6, n, prevRuneAt(input, 6))
	require.False(t, gotSecond)
}

func TestIsUnderlineTag_UTF8Letters_IntraWordRule(t *testing.T) {
	// "Ж_Ж": '_' is between two letters => should be plain text (not a tag)
	input := "Ж_Ж"
	n := len(input)

	underscoreIdx := len("Ж") // byte index after first UTF-8 rune
	require.Equal(t, byte('_'), input[underscoreIdx])

	got := isUnderlineTag(input, underscoreIdx, n, prevRuneAt(input, underscoreIdx))
	require.False(t, got)
}

func TestIsUnderlineTag_UTF8LeftBoundary_IsTag(t *testing.T) {
	// "Ж_": '_' has alnum on the left, but no right char => tag
	input := "Ж_"
	n := len(input)

	underscoreIdx := len("Ж")
	require.Equal(t, byte('_'), input[underscoreIdx])

	got := isUnderlineTag(input, underscoreIdx, n, prevRuneAt(input, underscoreIdx))
	require.True(t, got)
}

func TestActUnderline_ReturnsUnderlineToken(t *testing.T) {
	input := "a_b"
	i := 1 // underscore
	require.Equal(t, byte('_'), input[i])

	warns := make([]Warning, 0)

	tok, stride := actUnderline(input, i, &warns)

	require.Empty(t, warns)

	require.Equal(t, TypeUnderline, tok.Type)
	require.Equal(t, i, tok.Pos)
	require.Equal(t, 1, tok.Len)
	require.Equal(t, "_", tok.Val)

	require.Equal(t, 1, stride)
}
