package scum

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestASTText(t *testing.T) {
	d := testDict(t)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text",
			input: "just plain text",
			want:  "just plain text",
		},
		{
			name:  "nested tags and attribute",
			input: "pre [hello $$world$$]!url{https://example.com} post",
			want:  "pre hello world post",
		},
		{
			name:  "greedy code payload",
			input: "say `x * y` now",
			want:  "say x * y now",
		},
		{
			name:  "unicode text",
			input: "hé $$🙂$$",
			want:  "hé 🙂",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warns := newWarnings(t)
			ast := Parse(tt.input, &d, warns)

			require.Empty(t, warns.List())
			require.Equal(t, tt.want, ast.Text())
			require.Equal(t, len(tt.want), ast.TextByteLen)
		})
	}
}

func TestASTTextByteLenIsNotRuneCount(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	ast := Parse("hé $$🙂$$", &d, warns)

	require.Empty(t, warns.List())
	require.Equal(t, "hé 🙂", ast.Text())
	require.Equal(t, len("hé 🙂"), ast.TextByteLen)
	require.NotEqual(t, utf8.RuneCountInString("hé 🙂"), ast.TextByteLen)
}

func TestASTTextIncludesMismatchedClosingTagDemotedToText(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	ast := Parse("[*hi]", &d, warns)

	require.NotEmpty(t, warns.List())
	require.Equal(t, "hi]", ast.Text())
	require.Equal(t, len("hi]"), ast.TextByteLen)
}
