package scum

import (
	"fmt"
)

// Warning describes a recoverable problem found during tokenization or parsing.
type Warning struct {

	// Issue defines the type of the problem.
	Issue Issue

	// Pos defines the byte position in the input string at which the problem occurred.
	Pos int

	// TagID is the ID of the Tag causing the issue.
	TagID byte

	// CloseTagID is the ID of the Tag expected to close the Tag causing the issue.
	CloseTagID byte

	// Expected is the next expected symbol.
	Expected byte

	// Got is the symbol that was found instead of the expected one.
	Got byte
}

// WarningOverflowPolicy determines what happens when the maximum Warning capacity is reached.
type WarningOverflowPolicy int

const (
	// WarnOverflowNoCap means no limit for Warning recording.
	WarnOverflowNoCap WarningOverflowPolicy = iota

	// WarnOverflowNoRec means adding a new Warning is a no-op.
	WarnOverflowNoRec

	// WarnOverflowDrop means all Warnings after the overflow is reached are discarded.
	WarnOverflowDrop

	// WarnOverflowTrunc means all Warnings after the overflow is reached are discarded,
	// but the number of dropped warnings is recorded and a truncation Warning is added.
	WarnOverflowTrunc

	// should be the last. used for policy validation
	numPolicies
)

// Warnings maintains the list of issues found during tokenization or parsing.
// The list can have a maximum capacity, after which all further Warnings are
// discarded according to the configured [WarningOverflowPolicy].
type Warnings struct {
	policy WarningOverflowPolicy

	// list contains the Warnings
	list []Warning

	// maxWarnings defines how many Warnings the list can contain.
	// It prevents tokenization from stalling on inputs with huge numbers of issues.
	maxWarnings int

	// overflowed is true if the number of recorded Warnings reached the maximum capacity.
	overflowed bool

	// droppedCount is the number of discarded Warnings after the overflow.
	droppedCount int

	// firstDropPos is the input index from which Warnings are discarded.
	firstDropPos int
}

// IsOverflow reports whether the warning collector has reached its configured capacity.
func (w *Warnings) IsOverflow() bool {
	return w.overflowed
}

// DroppedCount is the number of Warnings discarded after overflow.
func (w *Warnings) DroppedCount() int {
	return w.droppedCount
}

// FirstDropPos is the index of the input from which the Warnings are discarded.
func (w *Warnings) FirstDropPos() int {
	return w.firstDropPos
}

// List returns the recorded Warnings.
func (w *Warnings) List() []Warning {
	return w.list
}

// Add appends a new [Warning] item to the inner list.
// If the policy is [WarnOverflowNoRec], this is no-op.
func (w *Warnings) Add(item Warning) {
	switch w.policy {
	case WarnOverflowNoRec:
		return
	case WarnOverflowNoCap:
		// TODO: unbounded append — consider pre-allocating or capping to reduce GC pressure
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
				Issue: IssueWarningsTruncated,
				Pos:   w.firstDropPos,
			})
		}
	}
	// Drop: do nothing else
}

// NewWarnings creates a Warnings collector with the given overflow policy and capacity.
// It returns a [ConfigError] if the policy is invalid or the cap is negative.
func NewWarnings(policy WarningOverflowPolicy, cap int) (Warnings, error) {

	if policy < 0 || policy > (numPolicies-1) {
		return Warnings{}, NewConfigError(
			IssueInvalidWarningsPolicy,
			fmt.Errorf("invalid warnings policy: %q", policy),
		)
	}

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
