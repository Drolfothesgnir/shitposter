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

	warns := make([]Warning, 0)

	tok, stride, ok := actCode(substr, 0, &warns)

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
	warns := make([]Warning, 0)

	tok, stride, ok := actCode(substr, 0, &warns)

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
	warns := make([]Warning, 0)

	tok, stride, ok := actCode(substr, 0, &warns)

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
	warns := make([]Warning, 0)

	tok, stride, ok := actCode(substr, 0, &warns)

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
	warns := make([]Warning, 0)

	tok, stride, ok := actCode(substr, 0, &warns)

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
	warns := make([]Warning, 0)

	tok, stride, ok := actCode(substr, 0, &warns)

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
	warns := make([]Warning, 0)

	tok, stride, ok := actCode(substr, 0, &warns)

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

func TestActCode_Block_AllowsLongerBacktickRunsInsideContent(t *testing.T) {
	// Opening/closing tag length is 3. Inside content we have 5 backticks in a row.
	// Per your logic, only a sequence with length == openTagLen closes the block,
	// so the 5-backtick run must be treated as content, not a closing tag.
	substr := "```print('hello `````')```"

	cur, width := utf8.DecodeRuneInString(substr)
	require.Equal(t, '`', cur)
	require.Equal(t, 1, width)

	warns := make([]Warning, 0)

	tok, stride, ok := actCode(substr, 0, &warns)

	require.True(t, ok)
	require.Empty(t, warns)

	require.Equal(t, TypeCodeBlock, tok.Type)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, len(substr), tok.Len)
	require.Equal(t, substr, tok.Val)

	require.Equal(t, len(substr), stride)
}

func TestActCode_Advanced(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedType   Type
		expectedVal    string
		expectedStride int
	}{
		{
			name:           "N+1 Rule: Inner sequence LONGER than tags",
			input:          "```print('hello `````')``` rest",
			expectedType:   TypeCodeBlock,
			expectedVal:    "```print('hello `````')```",
			expectedStride: 26,
		},
		{
			name:           "N+1 Rule: Inner sequence SHORTER than tags",
			input:          "```` code with `` ticks ````",
			expectedType:   TypeCodeBlock,
			expectedVal:    "```` code with `` ticks ````",
			expectedStride: 28,
		},
		{
			name:           "Edge Case: Backticks at the very start of content",
			input:          "````` ``` ```` `````", // 5 open, 3 content, 4 content, 5 close
			expectedType:   TypeCodeBlock,
			expectedVal:    "````` ``` ```` `````",
			expectedStride: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warns := make([]Warning, 0)
			token, stride, ok := actCode(tt.input, 0, &warns)

			require.True(t, ok)
			require.Equal(t, tt.expectedType, token.Type, "Type mismatch")
			require.Equal(t, tt.expectedVal, token.Val, "Value mismatch")
			require.Equal(t, tt.expectedStride, stride, "Stride mismatch")
		})
	}
}
