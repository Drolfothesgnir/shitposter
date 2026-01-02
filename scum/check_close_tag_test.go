package scum

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func dictStub(tags ...Tag) Dictionary {
	state := [256]Tag{}
	for _, t := range tags {
		state[t.Seq.ID()] = t
	}
	return Dictionary{tags: state}
}

// Optional but VERY useful: catches most future “relative vs absolute” regressions.
func requireCloseTagBoundsInvariants(t *testing.T, ctx *ActionContext) {
	t.Helper()

	require.NotNil(t, ctx)
	require.NotNil(t, ctx.Bounds)
	require.NotNil(t, ctx.Tag)
	require.NotNil(t, ctx.Dictionary)

	n := len(ctx.Input)
	contentStartIdx := ctx.Idx + ctx.Bounds.OpenWidth

	require.Equal(t, ctx.Idx, ctx.Bounds.Raw.Start)
	require.Equal(t, contentStartIdx, ctx.Bounds.Inner.Start)

	// CloseIdx contract: absolute index, -1 means “no candidate”
	if ctx.Bounds.CloseIdx == -1 {
		require.Equal(t, 0, ctx.Bounds.CloseWidth)
	} else {
		require.GreaterOrEqual(t, ctx.Bounds.CloseIdx, contentStartIdx)
		require.LessOrEqual(t, ctx.Bounds.CloseIdx, n)
		require.GreaterOrEqual(t, ctx.Bounds.CloseWidth, 0)
	}

	if ctx.Bounds.Closed {
		require.NotEqual(t, -1, ctx.Bounds.CloseIdx)
		require.Equal(t, ctx.Bounds.CloseIdx+ctx.Bounds.CloseWidth, ctx.Bounds.Raw.End)
		require.Equal(t, ctx.Bounds.CloseIdx, ctx.Bounds.Inner.End)
	} else {
		// Your current behavior for “not closed” is to span to end.
		require.Equal(t, n, ctx.Bounds.Raw.End)
		require.Equal(t, n, ctx.Bounds.Inner.End)
	}
}

func TestCheckCloseTag_NoClosingTagRegistered(t *testing.T) {
	// open tag at idx 0, open width = 2, no closing tag exists in dictionary
	d := dictStub()

	tag, err := NewTag([]byte{'$'}, "test", 0, 99)
	require.NoError(t, err)

	ctx := &ActionContext{
		Input:      "xxhello world",
		Idx:        0,
		Tag:        &tag,
		Dictionary: &d,
		Bounds:     &Bounds{OpenWidth: 2},
	}

	CheckCloseTag(ctx)

	require.False(t, ctx.Bounds.Closed)
	require.Equal(t, -1, ctx.Bounds.CloseIdx)
	require.Equal(t, 0, ctx.Bounds.CloseWidth)

	require.Equal(t, Span{Start: 0, End: len(ctx.Input)}, ctx.Bounds.Raw)
	require.Equal(t, Span{Start: 2, End: len(ctx.Input)}, ctx.Bounds.Inner)

	requireCloseTagBoundsInvariants(t, ctx)
}

func TestCheckCloseTag_ClosingTagContained_SingleByteClose(t *testing.T) {
	// input: "xxhello}tail"
	// contentStartIdx = 2
	// closing '}' is at absolute index 7
	input := "xxhello}tail"
	require.Equal(t, byte('}'), input[7])

	closeTag, err := NewTag([]byte{'}'}, "close_test", 0, 0)
	require.NoError(t, err)

	openTag, err := NewTag([]byte{'$'}, "open_test", 0, closeTag.ID())
	require.NoError(t, err)

	d := dictStub(closeTag)

	ctx := &ActionContext{
		Input:      input,
		Idx:        0,
		Tag:        &openTag,
		Dictionary: &d,
		Bounds:     &Bounds{OpenWidth: 2},
	}

	CheckCloseTag(ctx)

	require.True(t, ctx.Bounds.Closed)

	contentStartIdx := 2
	absCloseStart := 7
	closeW := 1

	require.Equal(t, absCloseStart, ctx.Bounds.CloseIdx)
	require.Equal(t, closeW, ctx.Bounds.CloseWidth)

	require.Equal(t, Span{Start: 0, End: absCloseStart + closeW}, ctx.Bounds.Raw)        // 0..8
	require.Equal(t, Span{Start: contentStartIdx, End: absCloseStart}, ctx.Bounds.Inner) // 2..7

	requireCloseTagBoundsInvariants(t, ctx)
}

func TestCheckCloseTag_ClosingTagContained_MultiByteClose(t *testing.T) {
	// input: "xxabc</>zzz"
	// contentStartIdx = 2
	// closing "</>" starts at absolute index 5, width 3
	input := "xxabc</>zzz"
	require.Equal(t, "</>", input[5:8])

	closeTag, err := NewTag([]byte("</>"), "close", 0, 0)
	require.NoError(t, err)

	openTag, err := NewTag([]byte{'$'}, "open", 0, closeTag.ID())
	require.NoError(t, err)

	d := dictStub(closeTag)

	ctx := &ActionContext{
		Input:      input,
		Idx:        0,
		Tag:        &openTag,
		Dictionary: &d,
		Bounds:     &Bounds{OpenWidth: 2},
	}

	CheckCloseTag(ctx)

	require.True(t, ctx.Bounds.Closed)

	contentStartIdx := 2
	absCloseStart := 5
	closeW := 3

	require.Equal(t, absCloseStart, ctx.Bounds.CloseIdx)
	require.Equal(t, closeW, ctx.Bounds.CloseWidth)

	require.Equal(t, Span{Start: 0, End: absCloseStart + closeW}, ctx.Bounds.Raw)        // 0..8
	require.Equal(t, Span{Start: contentStartIdx, End: absCloseStart}, ctx.Bounds.Inner) // 2..5

	requireCloseTagBoundsInvariants(t, ctx)
}

func TestCheckCloseTag_ClosingTagNotContained_ClosestPrefixReported(t *testing.T) {
	// input: "xxabc</xzzz"
	// contentStartIdx = 2
	// best candidate relStart=3 => absStart=5, matched="</" (len 2), but not fully closed
	input := "xxabc</xzzz"
	require.Equal(t, "</x", input[5:8])

	closeTag, err := NewTag([]byte("</>"), "close", 0, 0)
	require.NoError(t, err)

	openTag, err := NewTag([]byte{'$'}, "open", 0, closeTag.ID())
	require.NoError(t, err)

	d := dictStub(closeTag)

	ctx := &ActionContext{
		Input:      input,
		Idx:        0,
		Tag:        &openTag,
		Dictionary: &d,
		Bounds:     &Bounds{OpenWidth: 2},
	}

	CheckCloseTag(ctx)

	require.False(t, ctx.Bounds.Closed)

	absStart := 5
	matched := 2

	require.Equal(t, absStart, ctx.Bounds.CloseIdx)  // ABSOLUTE
	require.Equal(t, matched, ctx.Bounds.CloseWidth) // "</" matched

	// not closed => spans to end
	require.Equal(t, Span{Start: 0, End: len(input)}, ctx.Bounds.Raw)
	require.Equal(t, Span{Start: 2, End: len(input)}, ctx.Bounds.Inner)

	requireCloseTagBoundsInvariants(t, ctx)
}

func TestCheckCloseTag_ClosingTagNotContained_NoFirstByteFound(t *testing.T) {
	// input tail contains no '<' so IsContainedIn should return (false, -1, 0)
	input := "xxabcdefg"

	closeTag, err := NewTag([]byte("</>"), "close", 0, 0)
	require.NoError(t, err)

	openTag, err := NewTag([]byte{'$'}, "open", 0, closeTag.ID())
	require.NoError(t, err)

	d := dictStub(closeTag)

	ctx := &ActionContext{
		Input:      input,
		Idx:        0,
		Tag:        &openTag,
		Dictionary: &d,
		Bounds:     &Bounds{OpenWidth: 2},
	}

	CheckCloseTag(ctx)

	require.False(t, ctx.Bounds.Closed)
	require.Equal(t, -1, ctx.Bounds.CloseIdx)
	require.Equal(t, 0, ctx.Bounds.CloseWidth)

	require.Equal(t, Span{Start: 0, End: len(input)}, ctx.Bounds.Raw)
	require.Equal(t, Span{Start: 2, End: len(input)}, ctx.Bounds.Inner)

	requireCloseTagBoundsInvariants(t, ctx)
}
