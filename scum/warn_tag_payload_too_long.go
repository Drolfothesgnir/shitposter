package scum

// WarnTagPayloadTooLong adds [IssueTagPayloadTooLong] when payload scanning
// reached [Limits.MaxPayloadLen].
func WarnTagPayloadTooLong(ctx *ActionContext) {
	if !ctx.Bounds.PayloadLimitReached {
		return
	}

	ctx.Warns.Add(Warning{
		Issue: IssueTagPayloadTooLong,
		Pos:   ctx.Idx,
	})
}
