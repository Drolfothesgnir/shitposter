package scum

import (
	"fmt"
	"strconv"
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

	// otherwise set the escape action
	d.actions[char] = ActEscape
	return nil
}

func ActEscape(d *Dictionary, id byte, input string, i int, warns *[]Warning) (token Token, stride int, skip bool) {
	n := len(input)

	stride = 1

	// 1. Check if the escape symbol is redundant

	// 1.1 Check if the escape symbol is the very last char in the input

	// in this case add a Warning and skip current symbol
	if i+1 == n {
		*warns = append(*warns, Warning{
			Issue:       IssueUnexpectedEOL,
			Pos:         i,
			Description: "redundant escape symbol found at the very end of the input.",
		})

		skip = true
		return
	}

	// 1.2 Check if the next symbol is not special

	nextByte := input[i+1]

	// width of the next code-point
	nextWidth := 1

	// in this case we add a Warning of redundant escape
	if d.actions[nextByte] == nil {

		next := rune(nextByte)
		ok := true

		// if the next byte is a part of a multi-byte char,
		// extract the whole rune
		if nextByte > 127 {
			next, nextWidth, ok = extractNextRune(input[i+1:])
		}

		// define description based on whether extracted rune is an invalid symbol
		var got string
		if ok {
			got = strconv.QuoteRune(next)
		} else {
			got = "unknown symbol"
		}

		*warns = append(*warns, Warning{
			Issue: IssueRedundantEscape,
			Pos:   i + 1,
			Description: "expected any special symbol after the escape, at index " +
				strconv.Itoa(i+1) + ", found " + got + ".",
		})
	}

	// 2. Create Token
	token = Token{
		Type:  TokenEscapeSequence,
		TagID: id,
		Pos:   i,
		Width: 1 + nextWidth,
		Raw:   NewSpan(i, 1+nextWidth),
		Inner: NewSpan(i+1, nextWidth),
	}

	stride = 1 + nextWidth
	skip = false
	return
}
