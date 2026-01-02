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

func TestProbeStepCheckCloseTagGreedy_NoClosingTagRegistered(t *testing.T) {
	// input: open tag at idx 0, open width = 2, no closing tag exists in dictionary
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

	ok := ProbeStepCheckCloseTagGreedy(ctx)

	require.False(t, ok)
	require.False(t, ctx.Bounds.Closed)
	require.Equal(t, -1, ctx.Bounds.CloseIdx)
	require.Equal(t, 0, ctx.Bounds.CloseWidth)

	// Raw spans from opening tag start to end of input
	require.Equal(t, Span{Start: 0, End: len(ctx.Input)}, ctx.Bounds.Raw)

	// Inner spans from after opening tag to end of input
	require.Equal(t, Span{Start: 2, End: len(ctx.Input)}, ctx.Bounds.Inner)
}

func TestProbeStepCheckCloseTagGreedy_ClosingTagContained_SingleByteClose(t *testing.T) {
	// open tag starts at 0, open width = 2 -> contentStartIdx=2
	// closing tag sequence: "}" appears later
	input := "xxhello}tail"

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

	ok := ProbeStepCheckCloseTagGreedy(ctx)

	require.True(t, ok)
	require.True(t, ctx.Bounds.Closed)

	contentStartIdx := 2
	relCloseStart := 5 // "hello}..." => '}' at index 5 in "hello}tail"
	// NOTE: ProbeStep uses IsContainedIn on ctx.Input[contentStartIdx:], so startIdx is relative.
	// If you later switched to absolute CloseIdx, update expected accordingly.
	require.Equal(t, relCloseStart, ctx.Bounds.CloseIdx)
	require.Equal(t, 1, ctx.Bounds.CloseWidth)

	// Raw: from opening start (0) to end of closing tag (contentStartIdx + relCloseStart + w),
	// BUT your current implementation uses relative indexes directly (startIdx+w).
	// So expected is Span{0, relCloseStart+w} as per current code.
	require.Equal(t, Span{Start: 0, End: relCloseStart + 1}, ctx.Bounds.Raw)

	// Inner: from contentStartIdx to startIdx (relative), as per current code
	require.Equal(t, Span{Start: contentStartIdx, End: relCloseStart}, ctx.Bounds.Inner)
}

func TestProbeStepCheckCloseTagGreedy_ClosingTagContained_MultiByteClose(t *testing.T) {
	// closing tag is multi-byte, ensure w (width) is used
	input := "xxabc</>zzz" // contentStartIdx=2, close sequence "</>" starts at rel=3 in "abc</>zzz"

	// closeTag := Tag{Seq: mustSeq(t, '<', '/', '>')}

	closeTag, err := NewTag([]byte("</>"), "test", 0, 0)
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

	ok := ProbeStepCheckCloseTagGreedy(ctx)

	require.True(t, ok)
	require.True(t, ctx.Bounds.Closed)

	contentStartIdx := 2
	relCloseStart := 3
	closeW := 3

	require.Equal(t, relCloseStart, ctx.Bounds.CloseIdx)
	require.Equal(t, closeW, ctx.Bounds.CloseWidth)
	require.Equal(t, Span{Start: 0, End: relCloseStart + closeW}, ctx.Bounds.Raw)
	require.Equal(t, Span{Start: contentStartIdx, End: relCloseStart}, ctx.Bounds.Inner)
}

func TestProbeStepCheckCloseTagGreedy_ClosingTagNotContained_ClosestPrefixReported(t *testing.T) {
	// Here we want to ensure:
	// - contained=false
	// - Closed=false
	// - Raw/Inner go to end of input (greedy)
	// - CloseIdx and CloseWidth reflect the "closest alike" sequence (per IsContainedIn contract)
	//
	// Example: close sequence "</>" vs input "... </x" -> prefix "</" matched, then mismatch.
	input := "xxabc</xzzz" // contentStartIdx=2, candidate starts at rel=3 ("abc</xzzz"), prefix "</" len=2

	closeTag, err := NewTag([]byte("</>"), "test", 0, 0)
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

	ok := ProbeStepCheckCloseTagGreedy(ctx)

	require.False(t, ok)
	require.False(t, ctx.Bounds.Closed)

	// These expectations assume your fixed IsContainedIn returns:
	// (false, startIdxOfBestCandidate, matchedPrefixLen)
	relStart := 3
	matched := 2

	require.Equal(t, relStart, ctx.Bounds.CloseIdx)
	require.Equal(t, matched, ctx.Bounds.CloseWidth)

	// greedy: no full close => spans to end
	require.Equal(t, Span{Start: 0, End: len(input)}, ctx.Bounds.Raw)
	require.Equal(t, Span{Start: 2, End: len(input)}, ctx.Bounds.Inner)
}

func TestProbeStepCheckCloseTagGreedy_ClosingTagNotContained_NoFirstByteFound(t *testing.T) {
	// If the first byte of the close sequence is not present at all in the remaining input,
	// IsContainedIn should return (false, -1, 0) (or similar).
	input := "xxabcdefg" // no '<'

	closeTag, err := NewTag([]byte("</>"), "test", 0, 0)
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

	ok := ProbeStepCheckCloseTagGreedy(ctx)

	require.False(t, ok)
	require.False(t, ctx.Bounds.Closed)

	require.Equal(t, -1, ctx.Bounds.CloseIdx)
	require.Equal(t, 0, ctx.Bounds.CloseWidth)

	require.Equal(t, Span{Start: 0, End: len(input)}, ctx.Bounds.Raw)
	require.Equal(t, Span{Start: 2, End: len(input)}, ctx.Bounds.Inner)
}
