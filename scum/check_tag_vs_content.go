package scum

import "strings"

// CheckTagVsContent searches for the bounds of the Tag according to the [RuleTagVsContent].
func CheckTagVsContent(ctx *ActionContext) {
	n := len(ctx.Input)

	// 1. Count the opening sequence width
	char := ctx.Tag.ID()

	// 1 since the trigger char is already a part of the opening sequence
	openWidth := 1

	i := ctx.Idx

	for ; (openWidth < n-i) && (ctx.Input[i+openWidth] == char); openWidth++ {
	}

	// if the opening tag spans the entire rest of the string
	// the tag conisdered unclosed and the context is mutated accordingly
	if i+openWidth == n {
		mutateWithOnlyOpeningTag(ctx, openWidth)
		return
	}

	// 2. Find the closing sequence, discarding every sequence of the trigger char, which
	// is longer/shorter than opening width

	// we choose the next char after the first different char as the search start pos
	searchStartIdx := i + openWidth + 1

	for searchStartIdx < n-openWidth+1 {
		relIdx := strings.IndexByte(ctx.Input[searchStartIdx:], char)

		// if there are no closing sequence start in the rest of the input,
		// again mutate the context accordingly
		if relIdx == -1 {
			mutateWithOnlyOpeningTag(ctx, openWidth)
			return
		}

		// making relative to search start index absolute
		idx := searchStartIdx + relIdx

		// else calculate the current sequence's width

		// 1 since the trigger char is already a part of the closing sequence
		closeWidth := 1
		for ; (closeWidth < n-idx) && (ctx.Input[idx+closeWidth] == char); closeWidth++ {
		}

		// if width of the current sequence matches the opening width,
		// mutate the context and return
		if closeWidth == openWidth {
			ctx.Bounds.OpenWidth = openWidth
			ctx.Bounds.CloseIdx = idx
			ctx.Bounds.CloseWidth = closeWidth
			ctx.Bounds.Closed = true
			ctx.Bounds.Inner = Span{i + openWidth, idx}
			ctx.Bounds.Raw = Span{i, idx + closeWidth}
			return
		}

		// else continue searching from the next character after the
		// first different one
		searchStartIdx = idx + closeWidth + 1
	}

	// if no closing tag found in the end, change the context with only
	// opening Tag available
	mutateWithOnlyOpeningTag(ctx, openWidth)
}

func mutateWithOnlyOpeningTag(ctx *ActionContext, w int) {
	i := ctx.Idx
	n := len(ctx.Input)

	ctx.Bounds.CloseIdx = -1
	ctx.Bounds.CloseWidth = 0
	ctx.Bounds.Closed = false
	ctx.Bounds.OpenWidth = w
	ctx.Bounds.Inner = Span{i + w, n}
	ctx.Bounds.Raw = Span{i, n}
}
