package scum

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// isASCIIPrintable returns true if the byte is a printable ASCII character, that is
// its value is between 32 and 126.
func isASCIIPrintable(b byte) bool {
	// Printable ASCII characters are in the range 32 (space) to 126 (~)
	return b >= 32 && b <= 126
}

// checkByteDifference compares substr against the beginning of seq.
// It returns the index of the first differing byte, or -1 if no difference is found.
// substrShorter is true if substr is a prefix of seq but is shorter in length.
func checkByteDifference(substr string, seq []byte) (diffIndex int, substrShorter bool) {
	lenSubstr := len(substr)
	lenSeq := len(seq)

	diffIndex = -1
	substrShorter = lenSubstr < lenSeq

	minLen := min(lenSubstr, lenSeq)

	for i := range minLen {
		if substr[i] != seq[i] {
			diffIndex = i
			return
		}
	}

	return
}

// extractNextRune returns the first value (either simple ASCII or an UTF-8 code point) of the non-empty substr.
// It also returns the byte count of the found char and a bool flag, which is false in case the char is
// not a valid UTF-8 code point, but an [utf8.RuneError].
//
// WARNING: [utf8.DecodeRuneInString] returns width 0 if the decoded char is erroneous.
func extractNextRune(substr string) (next rune, width int, ok bool) {
	b := substr[0]

	// check if the first byte is simple ASCII
	if b < 128 {
		return rune(b), 1, true
	}

	// else we must decode the code point
	next, width = utf8.DecodeRuneInString(substr)
	ok = next != utf8.RuneError
	return
}

// checkTagBytes checks the provided [Tag.Seq] for being valid string representation
// and returns [ConfigError] if any issues occur.
func checkTagBytes(d *Dictionary, seq []byte) error {
	n := len(seq)

	// checking if the Tag's byte sequence is not empty
	if n == 0 {
		return NewConfigError(IssueInvalidTagSeqLen, errors.New("tag's byte sequence is empty"))
	}

	// checking if the Tag's byte sequence is not longer than [MaxTagLen]
	if n > MaxTagLen {
		return NewConfigError(IssueInvalidTagSeqLen,
			fmt.Errorf("expected the tag's byte sequence to be at most %d bytes long, got %d bytes", MaxTagLen, n))
	}

	// checking if the Tag is not a duplicate
	if d.actions[seq[0]] != nil {
		return NewConfigError(IssueDuplicateTagID,
			fmt.Errorf("special symbol with id %d already registered.", seq[0]))
	}

	// checking if the Tag consists of valid characters
	for i := range n {
		if !isASCIIPrintable(seq[i]) {
			return NewConfigError(IssueInvalidTagSeq,
				fmt.Errorf("unprintable character in the tag's byte sequence: %q at index %d", seq[i], i))
		}
	}

	return nil
}

// checkTagName checks the provided [Tag.Name] for being valid name and returns [ConfigError] if any issues occur.
func checkTagName(name string) error {
	// check if the name is not empty
	if name == "" {
		return NewConfigError(IssueInvalidTagNameLen, errors.New("tag's name is empty"))
	}

	// check if the name contains no more code points than [MaxTagNameLen]
	nameLen := utf8.RuneCountInString(name)

	if nameLen > MaxTagNameLen {
		return NewConfigError(IssueInvalidTagNameLen,
			fmt.Errorf("tag's name can be at most %d characters, but got %d", MaxTagNameLen, nameLen))
	}

	return nil
}
