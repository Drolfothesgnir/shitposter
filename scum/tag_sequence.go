package scum

import (
	"errors"
	"fmt"
)

// TagSequence maintains the Tag's string representation and its length.
type TagSequence struct {
	Bytes [MaxTagLen]byte
	Len   uint8
}

// ID returns the first byte of the Tag's byte sequence.
func (ts TagSequence) ID() byte { return ts.Bytes[0] }

// NewTagSequence creates a [TagSequence] from the provide series of bytes and possibly returns a [ConfigError].
func NewTagSequence(src ...byte) (TagSequence, error) {
	n := len(src)

	// check if the series is longer than allowed
	if n > MaxTagLen {
		return TagSequence{}, NewConfigError(IssueInvalidTagSeqLen,
			fmt.Errorf("tag's byte sequence can be at most %d bytes long, but got %d.", MaxTagLen, n))
	}

	// check if the series is empty
	if n == 0 {
		return TagSequence{},
			NewConfigError(IssueInvalidTagSeq, errors.New("provided Tag byte sequence is empty"))
	}

	var ts TagSequence

	for i, b := range src {
		// check if the series contains unprintable chracters
		if !isASCIIPrintable(b) {
			return TagSequence{}, NewConfigError(IssueInvalidTagSeq,
				fmt.Errorf("provided Tag byte sequence has unprintable character %q at index %d.", b, i))
		}

		ts.Bytes[i] = b
	}

	ts.Len = uint8(n)

	return ts, nil
}
