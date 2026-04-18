package scum

import (
	"fmt"
)

// TagSequence maintains the Tag's string representation and its length.
type TagSequence struct {
	Bytes [MaxTagLen]byte
	Len   uint8
}

// ID returns the first byte of the Tag's byte sequence.
func (seq TagSequence) ID() byte { return seq.Bytes[0] }

// IsContainedIn checks if src contains TagSequence. It returns true when contained,
// plus the start index and length of the sequence part that was found.
func (seq TagSequence) IsContainedIn(src string) (contained bool, startIdx, length int) {
	return longestCommonSubPrefix(src, seq.Bytes[:seq.Len])
}

// NewTagSequence creates a [TagSequence] from the provided byte sequence and possibly returns a [ConfigError].
func NewTagSequence(src []byte) (TagSequence, error) {
	n := len(src)

	// check if the series is longer than allowed
	if n > MaxTagLen {
		return TagSequence{}, NewConfigError(IssueInvalidTagSeqLen,
			fmt.Errorf("tag's byte sequence can be at most %d bytes long, but got %d.", MaxTagLen, n))
	}

	// check if the series is empty
	if n == 0 {
		return TagSequence{},
			newEmptySequenceError()
	}

	var ts TagSequence

	for i, b := range src {
		// check if the series contains unprintable characters
		if !isASCIIPrintable(b) {
			return TagSequence{}, NewConfigError(IssueUnprintableChar,
				fmt.Errorf("provided Tag byte sequence has unprintable character %q at index %d.", b, i))
		}

		ts.Bytes[i] = b
	}

	ts.Len = uint8(n)

	return ts, nil
}
