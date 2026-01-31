package scum

// Action is a function triggered by a special symbol defined in the [Dictionary].
// It processes the input string starting from the index i and returns a [Token], byte stride and
// a boolean flag which tells if the returned token is empty.
// WARNING: an Action MUST always return a stride > 0, even when skip = true.
type Action func(ac *ActionContext) (token Token, stride int, skip bool)

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

// Reset resets the Bounds to the initial state.
func (b *Bounds) Reset(i int) {
	b.CloseIdx = -1
	b.Closed = false
	b.Inner.Start = i
	b.Inner.End = i
	b.Raw.Start = i
	b.Raw.End = i + 1
	b.SeqValid = false
	b.Width = 1
	b.CloseWidth = 0
	b.KeyLenLimitReached = false
	b.PayloadLimitReached = false
}

// ActionContext defines the inter-step state for the Step execution.
type ActionContext struct {
	Tag        *Tag
	Dictionary *Dictionary
	State      *TokenizerState
	Warns      *Warnings
	Bounds     Bounds
	Trigger    byte
	Input      string
	Idx        int
	Token      Token
	Stride     int
	Skip       bool
}

// Reset resets the ActionContext to the initial state.
func (ac *ActionContext) Reset(char byte, i int) {
	ac.Tag = &ac.Dictionary.tags[char]
	ac.Idx = i
	ac.Token = Token{}
	ac.Stride = 0
	ac.Skip = false
	ac.Trigger = char
	ac.Bounds.Reset(i)
}

// NewActionContext creates new [ActionContext] based on provided [Dictionary], warnings slice, input,
// char/Tag ID and the current tokenizer position i.
func NewActionContext(d *Dictionary, s *TokenizerState, w *Warnings, input string, char byte, i int) ActionContext {
	b := Bounds{}
	b.Reset(i)

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
		Bounds:     b,
		Trigger:    char,
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
