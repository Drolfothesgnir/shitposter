package scum

// ProbeStepCheckCloseTagGreedy checks if the current [Greedy] [Tag] has it's closing counterpart somewhere in the
// input string, mutates the [ActionContext] accordingly and returns true if this is the case.
func ProbeStepCheckCloseTagGreedy(ctx *ActionContext) bool {
	n := len(ctx.Input)

	// 1. Determine if the closing Tag is even registered
	closeID := ctx.Tag.CloseID

	closeTag, exists := ctx.Dictionary.Tag(closeID)

	// contentStartIdx is the starting index of the plain text value of the Tag,
	// just after the opening Tag end
	contentStartIdx := ctx.Idx + ctx.Bounds.OpenWidth

	// if there is no valid closing Tag in the Dictionary, then we consider
	// the Tag unclosed and mutate the context accordingly
	if !exists {
		// we are telling that the current Tag is unclosed
		ctx.Bounds.Closed = false

		// also filling some values needed for creating Warnings
		// and some boundaries size calculations

		// we are telling that the closing Tag was not found
		ctx.Bounds.CloseIdx = -1

		// we are telling that the closing Tag sequence is empty
		ctx.Bounds.CloseWidth = 0

		// we are telling that the (unclosed) Tag spans the entire input after the
		// opening Tag's first byte
		ctx.Bounds.Raw = Span{ctx.Idx, n}

		// we are telling that the inner Tag's value spans the the entire input
		// after the opening tag
		ctx.Bounds.Inner = Span{contentStartIdx, n}
		return false
	}

	// if the closing Tag exists we check if it's contained in the rest of the input
	contained, startIdx, w := closeTag.Seq.IsContainedIn(ctx.Input[contentStartIdx:])

	// we are filling the closing Tag's info, found during the search
	ctx.Bounds.Closed = contained
	ctx.Bounds.CloseIdx = startIdx
	ctx.Bounds.CloseWidth = w

	// we mutate the Tag's bounds accordingly
	if contained {
		// if the closing Tag is completely contained in the rest of the input
		// we set Raw bound to be from the very first byte of the opening Tag,
		// to the last byte of the closing Tag
		ctx.Bounds.Raw = Span{ctx.Idx, startIdx + w}

		// and we set the Inner bound to the span between the opening and
		// closing Tags
		ctx.Bounds.Inner = Span{contentStartIdx, startIdx}
	} else {
		// if the closing Tag sequence is not fully present in the rest of
		// the input, we set Raw bound to be from the very first byte of
		// the opening Tag and the end of the string (according to the Greedy Tag definition)
		ctx.Bounds.Raw = Span{ctx.Idx, n}

		// and we set the Inner bound to the span from the start of the plain text to
		// the end of the string
		ctx.Bounds.Inner = Span{contentStartIdx, n}
	}

	return contained
}
