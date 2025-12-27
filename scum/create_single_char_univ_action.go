package scum

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// createSingleCharUnivAction decides which action to assign to the new Tag, based on [Tag.Greed].
// Returns [ConfigError] if the Greed level is invalid.
func createSingleCharUnivAction(t *Tag) (Action, error) {

	if t.Greed == NonGreedy {
		return createSingleCharUnivNonGreedyAction(t)
	}

	if t.Greed == Greedy {

	}

	if t.Greed == Grasping {

	}

	// guard case when the [Tag.Greed] level is invalid
	return nil, NewConfigError(
		IssueInvalidGreedLevel,
		fmt.Errorf("invalid Greed level occured during single-char universal Tag creation: %d; expected at most %d", t.Greed, MaxGreedLevel),
	)
}

func createSingleCharUnivNonGreedyAction(t *Tag) (Action, error) {
	if t.Rule == RuleNA {
		return createSingleCharUnivSimpleAction(t), nil
	}

	if t.Rule == RuleInfraWord {
		return createSingleCharUnivInfraWordAction(t), nil
	}

	return nil, NewConfigError(
		IssueInvalidRule,
		fmt.Errorf(
			"invalid Rule occured during single-char universal Tag creation: %d; available values are from 0 and up to %d",
			t.Rule, RuleInfraWord),
	)
}

func createSingleCharUnivSimpleAction(_ *Tag) Action {
	return func(d *Dictionary, id byte, input string, i int, prevRune rune, warns *[]Warning) (token Token, stride int, skip bool) {
		return singleByteTag(input, id, i)
	}
}

func isRealTag(input string, i int, id byte, prevRune rune) bool {
	n := len(input)

	// checking the left side
	leftIsAlphanum := false

	// if the symbol is not the first one
	if i > 0 {

		// if the previous symbol is the same as the Tag's, then the current one
		// is considered a plain text
		if rune(id) == prevRune {
			return false
		}

		leftIsAlphanum = unicode.IsLetter(prevRune) || unicode.IsDigit(prevRune)
	}

	// checking the right side
	rightIsAlphanum := false

	// if the symbol is not the last in the string

	var next rune

	if i+1 < n {
		// if the next symbol is 1-byte long and is
		if input[i+1] < 128 {
			next = rune(input[i+1])

			if rune(id) == next {
				return false
			}
		} else {
			next, _ = utf8.DecodeRuneInString(input[i+1:])
		}

		// if the next symbol is the same as the Tag's, then the current one
		// is considered a plain text

		rightIsAlphanum = unicode.IsLetter(next) || unicode.IsDigit(next)
	}

	return !leftIsAlphanum || !rightIsAlphanum
}

func createSingleCharUnivInfraWordAction(_ *Tag) Action {
	return func(d *Dictionary, id byte, input string, i int, prevRune rune, warns *[]Warning) (token Token, stride int, skip bool) {
		tagReal := isRealTag(input, i, id, prevRune)

		if tagReal {
			return singleByteTag(input, id, i)
		}

		skip = true

		return
	}
}
