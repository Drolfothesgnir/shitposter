package scum

// WarnUnclosedTag adds a [Warning] of an unclosed [Tag]. If the Tag is closed, is
// the closing Tag itself, or is unclosed because of payload limit reached, this is no-op.
func WarnUnclosedTag(ctx *ActionContext) {
	// if the Tag is closed, or is unclosed due to a limit reach, do nothing
	if ctx.Bounds.Closed || ctx.Tag.IsClosing() || ctx.Bounds.PayloadLimitReached {
		return
	}

	ctx.Warns.Add(Warning{
		Issue:      IssueUnclosedTag,
		Pos:        ctx.Idx,
		TagID:      ctx.Tag.ID(),
		CloseTagID: ctx.Tag.CloseID,
	})
}
