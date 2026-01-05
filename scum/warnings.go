package scum

import (
	"fmt"
)

// Warning describes the problem occured during the tokenizing or the parsing processes.
type Warning struct {

	// Issue defines the type of the problem.
	Issue Issue

	// Pos defines the byte position in the input string at which the problem occured.
	Pos int

	// Description is a human-readable story of what went wrong.
	Description string
}

// WarningOverflowPolicy determines what happends when the maximum Warnin capacity is reached.
type WarningOverflowPolicy int

const (
	// WarnNoCap means no limit for Warning recording.
	WarnOverflowNoCap WarningOverflowPolicy = iota

	// WarnNoRecording means adding new Warning is a no-op.
	WarnOverflowNoRec

	// WarnOverflowDrop means all Warnings, after the overflow reached, will be simply discarded.
	WarnOverflowDrop

	// WarnOverflowTrunc means all Warnings, after the overflow reached, will be discarded, but
	// the number dropped ones will be recorded and additional Warning, signalling the overflow,
	// added.
	WarnOverflowTrunc
)

// Warnings maintains the list of issues occured during the tokenization or the parsing.
// The list can have maximum capacity, after which all further Warnings will be discarded,
// with only number of discared ones available.
type Warnings struct {
	policy WarningOverflowPolicy

	// list contains the Warnings
	list []Warning

	// maxWarnings defines how many Warnings the list can contain.
	// It's used for prevent the stalling of the tokenization in case
	// of huge number of issues in the input.
	maxWarnings int

	// overflow is true if the number of recorded Warnings reached the maximum capacity
	overflowed bool

	// droppedCount is the number of the discarded Warnings after the overflow
	droppedCount int

	// firstDropPos is the index of the input from which the Warnings are discarded
	firstDropPos int
}

func (w *Warnings) IsOverflow() bool {
	return w.overflowed
}

// DroppedCount is a number of Warnings discarded after the overflow reach.
func (w *Warnings) DroppedCount() int {
	return w.droppedCount
}

// FirstDropPos is the index of the input from which the Warnings are discarded.
func (w *Warnings) FirstDropPos() int {
	return w.firstDropPos
}

func (w *Warnings) List() []Warning {
	return w.list
}

// Add appends new [Warning] item to the inner list.
// If the policy is [WarnOverflowNoRec], this is no-op.
func (w *Warnings) Add(item Warning) {
	switch w.policy {
	case WarnOverflowNoRec:
		return
	case WarnOverflowNoCap:
		w.list = append(w.list, item)
		return
	}

	// After overflow: Drop = ignore, Trunc = count + ignore
	if w.overflowed {
		if w.policy == WarnOverflowTrunc {
			w.droppedCount++
		}
		return
	}

	// capacity logic
	limit := w.maxWarnings
	if w.policy == WarnOverflowTrunc {
		limit = max(w.maxWarnings-1, 0) // reserve slot for truncation marker
	}

	if len(w.list) < limit {
		w.list = append(w.list, item)
		return
	}

	// First overflow happens now
	w.overflowed = true
	w.firstDropPos = item.Pos

	if w.policy == WarnOverflowTrunc {
		w.droppedCount = 1
		if w.maxWarnings > 0 {
			w.list = append(w.list, Warning{
				Issue:       IssueWarningsTruncated,
				Pos:         w.firstDropPos,
				Description: "too many warnings; further warnings suppressed",
			})
		}
	}
	// Drop: do nothing else
}

// NewWarnings creates a Warnings collector with the given overflow policy and capacity.
// It returns a ConfigError if cap is negative.
func NewWarnings(policy WarningOverflowPolicy, cap int) (Warnings, error) {

	if cap < 0 {
		return Warnings{}, NewConfigError(
			IssueNegativeWarningsCap,
			fmt.Errorf("warnings cap must be non-negative, got %d", cap),
		)
	}

	return Warnings{
		policy:      policy,
		list:        make([]Warning, 0, cap),
		maxWarnings: cap,
		overflowed:  false,
	}, nil
}
