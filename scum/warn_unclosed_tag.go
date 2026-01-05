package scum

import "strconv"

// WarnUnclosedTag adds a [Warning] of an unclosed [Tag]. If the Tag is closed or is
// the closing Tag itself, this is no-op.
func WarnUnclosedTag(ctx *ActionContext) {
	// if the Tag is closed, do nothing
	if ctx.Bounds.Closed || ctx.Tag.IsClosing() {
		return
	}

	desc := "unclosed tag " +
		strconv.Quote(ctx.Tag.Name) +
		": expected closing tag with ID " +
		strconv.Itoa(int(ctx.Tag.CloseID))

	*ctx.Warns = append(*ctx.Warns, Warning{
		Issue:       IssueUnclosedTag,
		Pos:         ctx.Idx,
		Description: desc,
	})
}
