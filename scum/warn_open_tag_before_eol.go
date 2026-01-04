package scum

import "strconv"

// WarnOpenTagBeforeEOL adds a [Warning] of the opening Tag before the very end of the input.
func WarnOpenTagBeforeEOL(ctx *ActionContext) {
	n := len(ctx.Input)
	w := ctx.Bounds.OpenWidth

	// if the Tag is not opening, or is not the last sequence in the input, do
	// nothing
	if ctx.Idx+w < n || !ctx.Tag.IsOpening() {
		return
	}

	// else add a Warning and skip the Tag as a plain text
	*ctx.Warns = append(*ctx.Warns, Warning{
		Issue: IssueUnexpectedEOL,
		Pos:   n,
		Description: "opening Tag with name " +
			strconv.Quote(ctx.Tag.Name) +
			" was found at the very end of the input and will be treated as plain text.",
	})
}
