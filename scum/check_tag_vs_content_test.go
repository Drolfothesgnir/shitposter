package scum

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTagVsContentCtx(t *testing.T, input string, idx int, ch byte) ActionContext {
	t.Helper()

	var d Dictionary
	warns, err := NewWarnings(WarnOverflowNoCap, 8)
	require.NoError(t, err)

	// Universal single-byte greedy tag using RuleTagVsContent.
	tag, err := NewTag([]byte{ch}, "CODE", ch, ch, WithGreed(Greedy), WithRule(RuleTagVsContent))
	require.NoError(t, err)

	d.tags[ch] = tag

	ctx := NewActionContext(&d, &warns, input, ch, idx)
	return ctx
}

func TestCheckTagVsContent_Closed_MatchingWidth(t *testing.T) {
	input := "&&&const T = a && b;&&&"
	ctx := newTagVsContentCtx(t, input, 0, '&')

	CheckTagVsContent(&ctx)

	require.True(t, ctx.Bounds.Closed)
	require.Equal(t, 3, ctx.Bounds.Width)
	require.Equal(t, 3, ctx.Bounds.CloseWidth)

	// closing run starts at the last "&&&"
	wantCloseIdx := strings.LastIndex(input, "&&&")
	require.Equal(t, wantCloseIdx, ctx.Bounds.CloseIdx)

	require.Equal(t, Span{0, wantCloseIdx + 3}, ctx.Bounds.Raw)
	require.Equal(t, Span{3, wantCloseIdx}, ctx.Bounds.Inner)
}

func TestCheckTagVsContent_IgnoresWrongWidthRuns(t *testing.T) {
	// Contains "&&" and "&&&&" in the content; only "&&&" should close.
	input := "&&&a && b &&&& c&&&"
	ctx := newTagVsContentCtx(t, input, 0, '&')

	CheckTagVsContent(&ctx)

	require.True(t, ctx.Bounds.Closed)
	require.Equal(t, 3, ctx.Bounds.Width)
	require.Equal(t, 3, ctx.Bounds.CloseWidth)

	wantCloseIdx := strings.LastIndex(input, "&&&")
	require.Equal(t, wantCloseIdx, ctx.Bounds.CloseIdx)

	require.Equal(t, Span{0, wantCloseIdx + 3}, ctx.Bounds.Raw)
	require.Equal(t, Span{3, wantCloseIdx}, ctx.Bounds.Inner)
}

func TestCheckTagVsContent_Unclosed_WhenOpeningSpansRest(t *testing.T) {
	input := "&&&"
	ctx := newTagVsContentCtx(t, input, 0, '&')

	CheckTagVsContent(&ctx)

	require.False(t, ctx.Bounds.Closed)
	require.Equal(t, 3, ctx.Bounds.Width)
	require.Equal(t, -1, ctx.Bounds.CloseIdx)
	require.Equal(t, 0, ctx.Bounds.CloseWidth)

	require.Equal(t, Span{0, len(input)}, ctx.Bounds.Raw)
	require.Equal(t, Span{3, len(input)}, ctx.Bounds.Inner) // empty inner
}

func TestCheckTagVsContent_Unclosed_WhenNoMatchingCloseRun(t *testing.T) {
	input := "&&&hello && world"
	ctx := newTagVsContentCtx(t, input, 0, '&')

	CheckTagVsContent(&ctx)

	require.False(t, ctx.Bounds.Closed)
	require.Equal(t, 3, ctx.Bounds.Width)
	require.Equal(t, -1, ctx.Bounds.CloseIdx)
	require.Equal(t, 0, ctx.Bounds.CloseWidth)

	require.Equal(t, Span{0, len(input)}, ctx.Bounds.Raw)
	require.Equal(t, Span{3, len(input)}, ctx.Bounds.Inner)
}

func TestCheckTagVsContent_OpeningNotAtStart(t *testing.T) {
	input := "xx&&&hi&&&yy"
	openIdx := 2
	ctx := newTagVsContentCtx(t, input, openIdx, '&')

	CheckTagVsContent(&ctx)

	require.True(t, ctx.Bounds.Closed)
	require.Equal(t, 3, ctx.Bounds.Width)
	require.Equal(t, 3, ctx.Bounds.CloseWidth)

	// Find the close after the opening.
	wantCloseIdx := strings.Index(input[openIdx+3:], "&&&")
	require.NotEqual(t, -1, wantCloseIdx)
	wantCloseIdx += openIdx + 3

	require.Equal(t, wantCloseIdx, ctx.Bounds.CloseIdx)
	require.Equal(t, Span{openIdx, wantCloseIdx + 3}, ctx.Bounds.Raw)
	require.Equal(t, Span{openIdx + 3, wantCloseIdx}, ctx.Bounds.Inner)
}

func TestCheckTagVsContent_AllSameChar_Unclosed(t *testing.T) {
	// With your current rule, openWidth becomes the whole run and thus "unclosed".
	input := "&&&&&&"
	ctx := newTagVsContentCtx(t, input, 0, '&')

	CheckTagVsContent(&ctx)

	require.False(t, ctx.Bounds.Closed)
	require.Equal(t, len(input), ctx.Bounds.Width)
	require.Equal(t, -1, ctx.Bounds.CloseIdx)
	require.Equal(t, Span{0, len(input)}, ctx.Bounds.Raw)
	require.Equal(t, Span{len(input), len(input)}, ctx.Bounds.Inner)
}
