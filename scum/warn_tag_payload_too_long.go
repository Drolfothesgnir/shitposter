package scum

func WarnTagPayloadTooLong(ctx *ActionContext) {
	if !ctx.Bounds.PayloadLimitReached {
		return
	}

	ctx.Warns.Add(Warning{
		Issue: IssueTagPayloadTooLong,
		Pos:   ctx.Idx,
	})
}
