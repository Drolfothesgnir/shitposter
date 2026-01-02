package scum

import (
	"errors"
	"fmt"
	"strings"
)

// TagSequence maintains the Tag's string representation and its length.
type TagSequence struct {
	Bytes [MaxTagLen]byte
	Len   uint8
}

// ID returns the first byte of the Tag's byte sequence.
func (seq TagSequence) ID() byte { return seq.Bytes[0] }

// IsContainedIn checks if the src contains TagSequence. It returnes the true if contained,
// start index of the possible Tag sequence and length of the sequence's part which is found.
// FIXME: it's all wrong. refactor it!
func (seq TagSequence) IsContainedIn(src string) (contained bool, startIdx, length int) {
	// looking for the start of the sequence in the src
	startIdx = strings.IndexByte(src, seq.Bytes[0])

	// if the sequence's start is not found, return immediately
	if startIdx == -1 {
		return false, -1, 0
	}

	seqLen := int(seq.Len)

	i := 1

	// checking if sequence's bytes and bytes after the first similar byte in the src are identical
	for i < seqLen-1 {
		if src[startIdx+i] != seq.Bytes[i] {
			// if there are some differences between two sequences return false, index of the tag-like secuence
			// start and a number of similar bytes
			return false, startIdx, i
		}

		i++
	}

	// else, the sequence is considered contained in the src, so we return true and stuff
	return true, startIdx, seqLen
}

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
