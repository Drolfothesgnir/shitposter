package scum

import (
	"strconv"
)

// WarnUnclosedTag adds a [Warning] of an unclosed [Tag]. If the Tag is closed, is
// the closing Tag itself, or is unclosed because of payload limit reached, this is no-op.
func WarnUnclosedTag(ctx *ActionContext) {
	// if the Tag is closed, or is unclosed due to a limit reach, do nothing
	if ctx.Bounds.Closed || ctx.Tag.IsClosing() || ctx.Bounds.PayloadLimitReached {
		return
	}

	desc := "unclosed tag " +
		strconv.Quote(ctx.Tag.Name) +
		": expected closing tag with ID " +
		strconv.Itoa(int(ctx.Tag.CloseID))

	ctx.Warns.Add(Warning{
		Issue:       IssueUnclosedTag,
		Pos:         ctx.Idx,
		Description: desc,
	})
}
