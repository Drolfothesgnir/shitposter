package scum

import (
	"fmt"
)

// SetEscapeTrigger sets the char as an escape symbol. Escaping means
// treating the next UTF-8 code point after the trigger as a plain text, whether
// it's special or not.
// NOTE: escape symbol can be only 1-byte long ASCII char.
func (d *Dictionary) SetEscapeTrigger(char byte) error {
	// if the char is not printable, abort with error
	if !isASCIIPrintable(char) {
		return NewConfigError(IssueUnprintableChar, fmt.Errorf("expected printable ASCII escape symbol, got: %q", char))
	}

	// if some action is already registered for this ID/char, abort with error
	if d.actions[char] != nil {
		return newDuplicateTagIDError(char)
	}

	// otherwise set the escape trigger and action
	d.escapeTrigger = char

	d.actions[char] = ActEscape
	return nil
}

func ActEscape(ac *ActionContext) (token Token, stride int, skip bool) {
	i := ac.Idx
	n := len(ac.Input)

	stride = 1

	// 1. Check if the escape symbol is redundant

	// 1.1 Check if the escape symbol is the very last char in the input

	// in this case add a Warning and skip current symbol
	if i+1 == n {
		ac.Warns.Add(Warning{
			Issue: IssueUnexpectedEOL,
			Pos:   i,
		})

		skip = true
		return
	}

	// 1.2 Check if the next symbol is not special

	nextByte := ac.Input[i+1]

	// width of the next code-point
	nextWidth := 1

	// in this case we add a Warning of redundant escape
	if ac.Dictionary.actions[nextByte] == nil {

		next := rune(nextByte)
		ok := true

		// if the next byte is a part of a multi-byte char,
		// extract the whole rune
		if nextByte > 127 {
			next, nextWidth, ok = extractNextRune(ac.Input[i+1:])
		}

		var gotByte byte
		if ok {
			gotByte = byte(next)
		}

		ac.Warns.Add(Warning{
			Issue: IssueRedundantEscape,
			Pos:   i,
			Got:   gotByte,
		})
	}

	// 2. Skip text
	token = Token{}
	stride = 1 + nextWidth
	skip = true
	return
}
