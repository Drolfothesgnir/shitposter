package scum

// CreateAction creates the specific action which is based on the Tag's features.
func CreateAction(t *Tag) Action {
	// allocating the new Plan on the heap
	p := new(Plan)

	// choose strategy based on whether the Tag is single char or multi char
	if t.Seq.Len == 1 {
		singleCharPlan(t, p)
	} else {
		multiCharPlan(t, p)
	}

	return func(ac *ActionContext) (token Token, stride int, skip bool) {
		return p.Run(ac)
	}
}

// singleCharPlan prepares the context with single byte opening Tag, and add
// steps based on the Tag's type, greed and rule.
func singleCharPlan(t *Tag, p *Plan) {
	// preparing to emit the single-byte opening Tag
	p.AddStep(MutateWith(PrepareSingleCharTag))

	switch {
	case t.IsUniversal():
		singleCharUniversalPlan(t, p)
	case t.IsOpening():
		singleCharOpeningPlan(t, p)
	case t.IsClosing():
		closingPlan(t, p)
	}
}

func multiCharPlan(t *Tag, p *Plan) {
	p.AddStep(MutateWith(CheckMultiCharTagSeq))
	p.AddStep(MutateWith(WarnInvalidSequence))
	p.AddStep(StepSkipInvalidTriggerTag)

	switch {
	case t.IsUniversal():
		multiCharUniversalPlan(t, p)
	case t.IsOpening():
		multiCharOpeningPlan(t, p)
	case t.IsClosing():
		closingPlan(t, p)
	}
}

func singleCharOpeningPlan(t *Tag, p *Plan) {
	p.AddStep(MutateWith(WarnOpenTagBeforeEOL))
	p.AddStep(StepSkipOpenTagBeforeEOL)

	singleCharUniversalPlan(t, p)
}

func singleCharUniversalPlan(t *Tag, p *Plan) {
	switch t.Greed {
	case NonGreedy:
		singleCharNonGreedyPlan(t, p)
	case Greedy:
		singleCharGreedyPlan(t, p)
	case Grasping:
		singleCharGraspingPlan(t, p)
	}
}

func closingPlan(_ *Tag, p *Plan) {
	p.AddStep(MutateWith(WarnCloseTagAfterStart))
	p.AddStep(StepSkipCloseTagAfterStart)
	p.AddStep(MutateWith(CountCloseTag))
	p.AddStep(StepEmitEntireTagToken)
}

// singleCharNonGreedyPlan creates plan for single-char tags
// with Greed 0 and optional Infra-Word rule.
func singleCharNonGreedyPlan(t *Tag, p *Plan) {
	// check the Infra-Word rule if neccessary
	if t.Rule == RuleInfraWord {
		p.AddStep(StepInfraWordCheck)
	}

	switch {
	case t.IsUniversal():
		p.AddStep(MutateWith(CountUniversalTag))
	case t.IsOpening():
		p.AddStep(MutateWith(CountOpenTag))
	}

	// emitting the opening Tag
	p.AddStep(StepEmitEntireTagToken)
}

func singleCharGreedyPlan(t *Tag, p *Plan) {
	addCloseTagCheck(t, p)
	p.AddStep(MutateWith(WarnUnclosedTag))
	p.AddStep(StepSkipUnclosedOpenTag)
	p.AddStep(MutateWith(CountTag))
	p.AddStep(StepEmitEntireTagToken)
}

func singleCharGraspingPlan(t *Tag, p *Plan) {
	addCloseTagCheck(t, p)
	p.AddStep(MutateWith(WarnUnclosedTag))
	p.AddStep(MutateWith(CountTag))
	p.AddStep(StepEmitEntireTagToken)
}

// addCloseTagCheck adds the appropriate close tag check based on the Tag's rule.
func addCloseTagCheck(t *Tag, p *Plan) {
	if t.Rule == RuleTagVsContent {
		p.AddStep(MutateWith(CheckTagVsContent))
		p.AddStep(MutateWith(WarnTagKeyTooLong))
	} else {
		p.AddStep(MutateWith(CheckCloseTag))
	}
	p.AddStep(MutateWith(WarnTagPayloadTooLong))
}

func multiCharUniversalPlan(t *Tag, p *Plan) {
	// if the Tag is greedy add closing tag check and an optional Warning
	if t.Greed > NonGreedy {
		p.AddStep(MutateWith(CheckCloseTag))
		p.AddStep(MutateWith(WarnUnclosedTag))
		p.AddStep(MutateWith(WarnTagPayloadTooLong))

		// only if the Tag is Greedy, add skip-if-unclosed
		if t.Greed == Greedy {
			p.AddStep(StepSkipUnclosedOpenTag)
		}
	}

	if t.Greed == NonGreedy {
		switch {
		case t.IsUniversal():
			p.AddStep(MutateWith(CountUniversalTag))
		case t.IsOpening():
			p.AddStep(MutateWith(CountOpenTag))
		}
	} else {
		p.AddStep(MutateWith(CountTag))
	}

	p.AddStep(StepEmitEntireTagToken)
}

func multiCharOpeningPlan(t *Tag, p *Plan) {
	p.AddStep(MutateWith(WarnOpenTagBeforeEOL))
	p.AddStep(StepSkipOpenTagBeforeEOL)

	// except checking the opening-before-EOL case,
	// the opening plan is the same as universal
	multiCharUniversalPlan(t, p)
}

func CountTag(ac *ActionContext) {
	ac.State.TagsTotal++
}

func CountCloseTag(ac *ActionContext) {
	CountTag(ac)
	ac.State.CloseTags++
}

func CountOpenTag(ac *ActionContext) {
	CountTag(ac)
	ac.State.OpenTags++
}

func CountUniversalTag(ac *ActionContext) {
	CountTag(ac)
	ac.State.UniversalTags++
}
