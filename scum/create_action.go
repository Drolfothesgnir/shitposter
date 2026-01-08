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

	return func(d *Dictionary, id byte, input string, i int, warns *Warnings) (token Token, stride int, skip bool) {
		ctx := NewActionContext(d, warns, input, id, i)

		return p.Run(&ctx)
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
		singleCharClosingPlan(t, p)
	}
}

func multiCharPlan(t *Tag, p *Plan) {
	// do stuff
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

func singleCharOpeningPlan(t *Tag, p *Plan) {
	p.AddStep(MutateWith(WarnOpenTagBeforeEOL))
	p.AddStep(StepOpenTagBeforeEOL)

	switch t.Greed {
	case NonGreedy:
		singleCharNonGreedyPlan(t, p)
	case Greedy:
		singleCharGreedyPlan(t, p)
	case Grasping:
		singleCharGraspingPlan(t, p)
	}
}

func singleCharClosingPlan(_ *Tag, p *Plan) {
	p.AddStep(MutateWith(WarnCloseTagAfterStart))
	p.AddStep(StepCloseTagAfterStart)
	p.AddStep(StepEmitEntireTagToken)
}

// singleCharNonGreedyPlan creates plan for single-char tags
// with Greed 0 and optional Infra-Word rule.
func singleCharNonGreedyPlan(t *Tag, p *Plan) {
	// check the Infra-Word rule if neccessary
	if t.Rule == RuleInfraWord {
		p.AddStep(StepInfraWordCheck)
	}

	// emitting the opening Tag
	p.AddStep(StepEmitEntireTagToken)
}

func singleCharGreedyPlan(t *Tag, p *Plan) {
	// if the Tag-Vs-Content rule applies check it
	if t.Rule == RuleTagVsContent {
		p.AddStep(MutateWith(CheckTagVsContent))
	} else {
		// else simply check if the Tag is closed
		p.AddStep(MutateWith(CheckCloseTag))
	}

	// if the Tag is unclosed, add a Warning and skip it as text

	p.AddStep(MutateWith(WarnUnclosedTag))

	p.AddStep(StepSkipUnclosedOpenTag)

	// otherwise emit the tag
	p.AddStep(StepEmitEntireTagToken)
}

func singleCharGraspingPlan(t *Tag, p *Plan) {
	// if the Tag-Vs-Content rule applies check it
	if t.Rule == RuleTagVsContent {
		p.AddStep(MutateWith(CheckTagVsContent))
	} else {
		// else simply check if the Tag is closed
		p.AddStep(MutateWith(CheckCloseTag))
	}

	// Warning will appear if the Tag is unclosed, but the entire Tag
	// will be emitted anyway
	p.AddStep(MutateWith(WarnUnclosedTag))

	p.AddStep(StepEmitEntireTagToken)
}
