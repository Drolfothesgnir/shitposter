package scum

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func mustTag(t *testing.T, seq []byte, name string, openID, closeID byte, opts ...TagDecorator) Tag {
	tag, err := NewTag(seq, name, openID, closeID, opts...)
	require.NoError(t, err)
	return tag
}

// helper to build ActionContext for StepInfraWordCheck
func mkCtx(input string, idx int, tag *Tag) *ActionContext {
	return &ActionContext{
		Input: input,
		Idx:   idx,
		Tag:   tag,
		// the step should set these itself when it returns true
		Stride: 0,
		Skip:   false,
	}
}

func TestStepInfraWordCheck_TagAtBeginning_IsRealTag(t *testing.T) {
	// idx=0 -> leftIsWordPart=false => should treat as real tag
	tag := mustTag(t, []byte{'#'}, "HASH", 0, 0)

	ctx := mkCtx("#abc", 0, &tag)

	ok := StepInfraWordCheck(ctx)

	require.False(t, ok)
	require.False(t, ctx.Skip)
	require.Equal(t, 0, ctx.Stride)
}

func TestStepInfraWordCheck_TagAtEnd_IsRealTag(t *testing.T) {
	// idx = last -> rightIsWordPart=false => real tag
	tag := mustTag(t, []byte{'#'}, "HASH", 0, 0)

	ctx := mkCtx("abc#", 3, &tag)

	ok := StepInfraWordCheck(ctx)

	require.False(t, ok)
	require.False(t, ctx.Skip)
	require.Equal(t, 0, ctx.Stride)
}

func TestStepInfraWordCheck_SurroundedByASCIIAlphanum_IsPlainText(t *testing.T) {
	// a#a -> both sides are ASCII alphanum => infra-word => plain text
	tag := mustTag(t, []byte{'#'}, "HASH", 0, 0)

	ctx := mkCtx("a#a", 1, &tag)

	ok := StepInfraWordCheck(ctx)

	require.True(t, ok)
	require.True(t, ctx.Skip)
	require.Equal(t, 1, ctx.Stride)
}

func TestStepInfraWordCheck_LeftAlphanum_RightSpace_IsRealTag(t *testing.T) {
	// a#  -> right is not word part => real tag
	tag := mustTag(t, []byte{'#'}, "HASH", 0, 0)

	ctx := mkCtx("a# ", 1, &tag)

	ok := StepInfraWordCheck(ctx)

	require.False(t, ok)
	require.False(t, ctx.Skip)
	require.Equal(t, 0, ctx.Stride)
}

func TestStepInfraWordCheck_LeftSpace_RightAlphanum_IsRealTag(t *testing.T) {
	//  #a -> left is not word part => real tag
	tag := mustTag(t, []byte{'#'}, "HASH", 0, 0)

	ctx := mkCtx(" #a", 1, &tag)

	ok := StepInfraWordCheck(ctx)

	require.False(t, ok)
	require.False(t, ctx.Skip)
	require.Equal(t, 0, ctx.Stride)
}

func TestStepInfraWordCheck_SurroundedByASCIIPunct_IsPlainText(t *testing.T) {
	// -#- -> hyphen is ASCIIPunct in your definition => infra-word => plain text
	tag := mustTag(t, []byte{'#'}, "HASH", 0, 0)

	ctx := mkCtx("-#-", 1, &tag)

	ok := StepInfraWordCheck(ctx)

	require.True(t, ok)
	require.True(t, ctx.Skip)
	require.Equal(t, 1, ctx.Stride)
}

func TestStepInfraWordCheck_SurroundedBySameTrigger_IsPlainText(t *testing.T) {
	// ### at idx=1 -> left and right are trigger byte => infra-word => plain text
	tag := mustTag(t, []byte{'#'}, "HASH", 0, 0)

	ctx := mkCtx("###", 1, &tag)

	ok := StepInfraWordCheck(ctx)

	require.True(t, ok)
	require.True(t, ctx.Skip)
	require.Equal(t, 1, ctx.Stride)
}

func TestStepInfraWordCheck_LeftUnicodeLetter_RightASCIIAlphanum_IsPlainText(t *testing.T) {
	// å#b : left rune is unicode letter, right is ASCII alphanum => infra-word => plain text
	tag := mustTag(t, []byte{'#'}, "HASH", 0, 0)

	input := "å#b"
	// bytes: 'å' is two bytes, '#' at index 2
	ctx := mkCtx(input, 2, &tag)

	ok := StepInfraWordCheck(ctx)

	require.True(t, ok)
	require.True(t, ctx.Skip)
	require.Equal(t, 1, ctx.Stride)
}

func TestStepInfraWordCheck_LeftASCIIAlphanum_RightUnicodeLetter_IsPlainText(t *testing.T) {
	// a#Ж : right rune is unicode letter => infra-word => plain text
	tag := mustTag(t, []byte{'#'}, "HASH", 0, 0)

	input := "a#Ж"
	// bytes: 'a'(0), '#'(1), 'Ж' starts at index 2
	ctx := mkCtx(input, 1, &tag)

	ok := StepInfraWordCheck(ctx)

	require.True(t, ok)
	require.True(t, ctx.Skip)
	require.Equal(t, 1, ctx.Stride)
}

func TestStepInfraWordCheck_LeftUnicodeLetter_RightSpace_IsRealTag(t *testing.T) {
	// Ж#  -> left is unicode letter, right is space => real tag
	tag := mustTag(t, []byte{'#'}, "HASH", 0, 0)

	input := "Ж# "
	// 'Ж' is two bytes, '#' at index 2
	ctx := mkCtx(input, 2, &tag)

	ok := StepInfraWordCheck(ctx)

	require.False(t, ok)
	require.False(t, ctx.Skip)
	require.Equal(t, 0, ctx.Stride)
}

func TestStepInfraWordCheck_LeftSpace_RightUnicodeLetter_IsRealTag(t *testing.T) {
	//  #Ж -> left is space => real tag
	tag := mustTag(t, []byte{'#'}, "HASH", 0, 0)

	input := " #Ж"
	// '#' at index 1
	ctx := mkCtx(input, 1, &tag)

	ok := StepInfraWordCheck(ctx)

	require.False(t, ok)
	require.False(t, ctx.Skip)
	require.Equal(t, 0, ctx.Stride)
}
