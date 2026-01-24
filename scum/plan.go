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

	// Width is the length in bytes of the ACTUAL trigger Tag sequence,
	// found in the input.
	Width int

	// CloseWidth is the length in bytes of the ACTUAL closing Tag sequence,
	// found in the input. It's only relevant for the greedy opening and
	// universal Tags, since a closing Tag can't have its own closing Tag.
	//
	// WARNING: use it only for get the complementary closing Tag's width.
	// For the trigger Tag's width use Width.
	CloseWidth int

	// CloseIdx is the index of the start of the closing Tag sequence.
	CloseIdx int

	// SeqValid is true when the found Tag's byte sequence is completed.
	// Useful for multi-char Tags.
	SeqValid bool

	// KeyLenLimitReached is true when the opening or the closing Tag sequence
	// is longer than [Limits.MaxKeyLen].
	KeyLenLimitReached bool

	// PayloadLimitReached is true when the Tag's or attribute's payload is
	// longer then [Limits.]
	PayloadLimitReached bool
}

// NewBounds creates new [Bounds] based on the index i.
func NewBounds(i int) Bounds {
	return Bounds{
		Raw:        NewSpan(i, 1),
		Inner:      NewSpan(i, 0),
		Closed:     false,
		Width:      1,
		CloseWidth: 0,
		CloseIdx:   -1,
		SeqValid:   false,
	}
}

// ActionContext defines the inter-step state for the Step execution.
type ActionContext struct {
	Tag        *Tag
	Dictionary *Dictionary
	State      *TokenizerState
	Input      string
	Idx        int
	Token      Token
	Stride     int
	Skip       bool
	Warns      *Warnings
	Bounds     *Bounds
}

// NewActionContext creates new [ActionContext] based on provided [Dictionary], warnings slice, input,
// char/Tag ID and the current tokenizer position i.
func NewActionContext(d *Dictionary, s *TokenizerState, w *Warnings, input string, char byte, i int) ActionContext {
	b := NewBounds(i)

	// since the ActionContext is only called in the actual Action, we assume the required Tag is
	// available.
	t := &d.tags[char]

	return ActionContext{
		Tag:        t,
		Dictionary: d,
		State:      s,
		Input:      input,
		Idx:        i,
		Warns:      w,
		Token:      Token{},
		Stride:     0,
		Skip:       false,
		Bounds:     &b,
	}
}

// Step is a function which is a part of the Action, responsibe for the one single part.
// Step returns true when it can handle the case completely.
type Step func(*ActionContext) bool

// Plan maintains sequence of steps and organizes their execution.
type Plan struct {
	Steps []Step
}

// Run executes steps sequentially.
// WARNING: Steps should not rely on previously set Token, Stride and Skip, since those are
// reset on every new Step start.
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

// AddStep appends new [Step] to the [Plan].
func (p *Plan) AddStep(s Step) {
	p.Steps = append(p.Steps, s)
}

// Mutator is a function which recieves [ActionContext], does some manipulations with it,
// like checks or preparations, and modifies it's state.
type Mutator func(*ActionContext)

// MutateWith makes a [Step] out of a [Mutator] function. It's used to allow mutators
// as legitimate [Plan] steps, which never handle the current case, but only perform
// some jobs on the [ActionContext].
func MutateWith(m Mutator) Step {
	return func(ac *ActionContext) bool {
		// do the job
		m(ac)

		// never handle the current case
		return false
	}
}
