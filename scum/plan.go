package scum

// Bounds contains the metadata about current Tag's string representation's
// consistency.
// Useful for creating Warnings.
type Bounds struct {
	// Raw defines the start and exclusive end indexes of the entire Tag's value.
	// Useful for handling greedy Tags.
	Raw Span

	// Inner defines the start and exclusive end indexes of the Tag's text value.
	// Useful for handling greedy Tags.
	Inner Span

	// Closed is true if the current Tag is properly closed.
	Closed bool

	// OpenWidth is the length in bytes of the ACTUAL opening Tag sequence,
	// found in the input.
	OpenWidth int

	// CloseWidth is the length in bytes of the ACTUAL closing Tag sequence,
	// found in the input.
	CloseWidth int

	// CloseIdx is the index of the start of the closing Tag sequence.
	CloseIdx int
}

// ActionContext defines the inter-step state for the Step execution.
type ActionContext struct {
	Tag        *Tag
	Dictionary *Dictionary
	Input      string
	Idx        int
	Token      Token
	Stride     int
	Skip       bool
	Warns      *[]Warning
	Bounds     *Bounds
}

// Step is a function which is a part of the Action, responsibe for the one single part.
// Step returns true when it can handle the case completely.
type Step func(*ActionContext) bool

// Plan maintains sequence of steps and organizes their execution.
type Plan struct {
	Steps []Step
}

// Run executes steps sequentially.
func (p Plan) Run(ctx *ActionContext) (Token, int, bool) {
	for _, s := range p.Steps {
		// cleaning the context at the start of the new Step
		ctx.Token = Token{}
		ctx.Stride = 0
		ctx.Skip = false

		// if the current Step handles the case successfully we stop the execution and
		// return the Step's result.
		if s(ctx) {
			return ctx.Token, ctx.Stride, ctx.Skip
		}

		// else we continue looking for the Step which can handle the case
	}

	// if no Steps succeeded, we just move to the next character
	return Token{}, 1, true
}
