// S.C.U.M. stands for Shitposter's Completely User-customizable Markup.
// You can set opening, closing, universal and greedy tags along with the escape symbol dynamically during runtime.
// Tags will have the tag name of your choice and the end AST will be built based on it.
//
// WARNING: It works exclusively with simple 1-byte long ASCII symbols as tags.
package scum

import (
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"
)

type TokenType int

const (
	TokenText TokenType = iota
	TokenOpeningTag
	TokenClosingTag
	TokenUniversalTag
	TokenGreedyTag
	TokenEscapeSequence
)

const MaxTagLength = 4

// Token is the result of the first stage processing of a part of the input string.
// It contains metadata and value of the processed sequence of bytes.
type Token struct {
	// Name is a User-defined human-readable ID of the tag.
	Name string

	// Type defines the type of the Token, e.g. opening, closing, or universal tag, or an escape sequence.
	Type TokenType

	// TagID a unique leading byte of the tag byte sequence, defined by the User.
	TagID byte

	// OpeningTagID is useful when the token is of type [TokenClosingTag], to help the Parser recognize the next steps,
	// and to check if the open/close behaviour is consistent.
	//
	// Example: current top opening tag in the Parser's Internal State Stack is with ID 0x3c ('<' sign) and
	// it's corresponding closing tag has ID 0x3e ('>' sign), but the Parser should not know it. Imagine the next token in
	// the stream has OpeningTagID 0x5b ('[' sign). The Parser will see the inconsistency between PISS's top tag ID and the
	// next token opening tag ID and will act accordingly.
	OpeningTagID byte

	// Pos defines the starting byte position of the tag's sequence in the input string.
	Pos int

	// Width defines count of bytes in the tag's sequence.
	//
	// Example: Imagine, you have defined a universal tag with name 'BOLD' and a byte sequence of "$$". The sequence has
	// 2 bytes in it, 1 per each '$', so the corresponding token will have width of 2.
	Width int

	// Raw defines the substring associated with the tag's value including both tag strings and the inner plain text.
	//
	// Example: Imagine, you have defined a greedy tag with name 'URL' and a pattern like this: "(...)", where
	// '(' is the opening tag and the ')' is the closing tag. When interpreting string "(https://some-address.com)",
	// the Raw field will consist of the entire matched string, that is the "(https://some-address.com)".
	// For Text tokens Raw and Inner fields are the same.
	Raw string

	// Inner defines the plain text, in case of token with type [TokenText], or the matched string, stripped of tags.
	//
	// Example: Imagine, you have defined a greedy tag with name 'URL' and a pattern like this: "(...)", where
	// '(' is the opening tag and the ')' is the closing tag. When interpreting string "(https://some-address.com)",
	// the Inner field will consist of only the "https://some-address.com".
	Inner string
}

// Issue defines types of problems we might encounter during the tokenizing or the parsing processes.
type Issue int

const (
	IssueUnexpectedEOL Issue = iota
	IssueUnexpectedSymbol
	IssueMisplacedClosingTag
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

// Action is a function triggered by a special symbol defined in the [Dictionary].
// It processes the input string strating from the index i and returns a [Token] and,
// possibly, adds a [Warning].
type Action func(input string, i int, warns *[]Warning) (token Token, stride int)

type Dictionary struct {
	Actions [256]Action
}

// Use it like d.AddOpenTag("BOLD", '$', '$')
func (d *Dictionary) AddOpeningTag(name string, openSeq ...byte) error {
	l := len(openSeq)
	if l > MaxTagLength {
		return fmt.Errorf("Opening tag sequence is too long: expected at most %d symbols, got %d.", MaxTagLength, l)
	}

	if l == 0 {
		return errors.New("No bytes provided for the opening tag sequence.")
	}

	firstByte := openSeq[0]

	if d.Actions[firstByte] != nil {
		return fmt.Errorf("Action with trigger symbol %q already exist. Remove it manually before setting a new one.", firstByte)
	}

	if l > 1 {
		d.Actions[firstByte] = createOpenTagActionMultiple(name, openSeq)
	} else {
		d.Actions[firstByte] = createOpenTagActionSingle(name, firstByte)
	}

	return nil
}

// createOpenTagActionSingle creates an [Action] for an interpretation of a single, 1-byte long opening tag.
// It returns a [Token] with type [TokenOpeningTag], if the trigger symbol is not the
// last in the input string, and a token with type [TokenText], along with adding a [Warning] otherwise.
func createOpenTagActionSingle(name string, char byte) Action {
	return func(input string, i int, warns *[]Warning) (token Token, stride int) {
		// string representation of the opening tag
		tag := input[i : i+1]

		// default happy params
		t := TokenOpeningTag
		inner := ""

		// if the trigger symbol is the last in the input string, return the [TokenText] token and add a
		// [Warning]
		if i+1 == len(input) {
			t = TokenText
			inner = tag

			desc := "Unexpected end of the line after the opening tag '" +
				tag + "', while interpreting the opening tag with name '" + name + "'."

			*warns = append(*warns, Warning{
				Issue:       IssueUnexpectedEOL,
				Pos:         i + 1,
				Description: desc,
			})
		}

		// happy case
		token = Token{
			Name:  name,
			Type:  t,
			TagID: char,
			Pos:   i,
			Width: 1,
			Raw:   tag,
			Inner: inner,
		}

		// in any case we process exactly 1 byte
		stride = 1
		return
	}
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

// createOpenTagMultiple creates an Action for a new opening tag with the provided name, which starts with sequence of bytes - seq.
func createOpenTagActionMultiple(name string, seq []byte) Action {
	return func(input string, i int, warns *[]Warning) (token Token, stride int) {
		n := len(input)

		// 1. Checking the opening tag consistency

		// 1.1 Checking if the input after index i is similar to the rest of the opening tag byte sequence
		// we start comparing from i+1 because byte at index i is the same, since it was an Action trigger
		diffIndex, substrShorter := checkByteDifference(input[i+1:], seq[1:])

		// if the input's opening sequence is different from seq, that is diffIndex > -1, return [Token] with
		// type [TokenText] and add a Warning
		if diffIndex > -1 {
			// adjusting the relative diffIndex
			absDiffIndex := diffIndex + i + 1

			matchedSeq := input[i:absDiffIndex]

			token = Token{
				Name:  name,
				Type:  TokenText,
				TagID: seq[0],
				Pos:   i,
				Width: len(matchedSeq),
				Raw:   matchedSeq,
				Inner: matchedSeq,
			}

			wrong, _, ok := extractNextRune(input[absDiffIndex:])

			quoted := strconv.QuoteRune(wrong)

			got := ""

			if !ok {
				got = "unrecognizable character "
			}

			desc := "Unexpected symbol at index " +
				strconv.Itoa(absDiffIndex) +
				" while interpreting the opening tag with name '" +
				name + "': expected to get '" + string(seq[diffIndex+1]) +
				"', got " + got + quoted + "."

			*warns = append(*warns, Warning{
				Issue:       IssueUnexpectedSymbol,
				Pos:         absDiffIndex,
				Description: desc,
			})

			// we've processed all bytes from the index i and up to index of the first divergence
			stride = absDiffIndex - i
			return
		}

		// 1.2 Checking if the input string ended before completing the opening tag sequence, that is substrShorter == true

		// in this case we return the unfinished tag as [Token] with type [TokenText], and a [Warning]
		if substrShorter {
			matchedSeq := input[i:n]

			matchedSeqLen := len(matchedSeq)

			token = Token{
				Name:  name,
				Type:  TokenText,
				TagID: seq[0],
				Pos:   i,
				Width: matchedSeqLen,
				Raw:   matchedSeq,
				Inner: matchedSeq,
			}

			// seq[n-i:] is a sub sequence of seq starting from the first missing byte from the string
			desc := "Unexpected end of the line while interpreting the opening tag with name '" +
				name + "': expected to get '" + string(seq[n-i:]) + "' but got EOL."

			*warns = append(*warns, Warning{
				Issue:       IssueUnexpectedEOL,
				Pos:         n,
				Description: desc,
			})

			// we've processed all bytes from i to the end of the input
			stride = matchedSeqLen
			return
		}

		// 2. Happy case
		seqLen := len(seq)

		token = Token{
			Name:  name,
			Type:  TokenOpeningTag,
			TagID: seq[0],
			Pos:   i,
			Width: seqLen,
			Raw:   input[i : i+seqLen],
			// Leaving Inner empty since it's not a tag with text inside
		}

		return
	}
}

func (d *Dictionary) AddClosingTag(name string, openTagID byte, closeSeq ...byte) error {
	l := len(closeSeq)
	if l > MaxTagLength {
		return fmt.Errorf("Closing tag sequence is too long: expected at most %d symbols, got %d.", MaxTagLength, l)
	}

	if l == 0 {
		return errors.New("No bytes provided for the closing tag sequence.")
	}

	firstByte := closeSeq[0]

	if d.Actions[firstByte] != nil {
		return fmt.Errorf("Action with trigger symbol %q already exist. Remove it manually before setting a new one.", firstByte)
	}

	// if l > 1 {
	// 	d.Actions[firstByte] = createOpenTagActionMultiple(name, openSeq)
	// } else {
	// 	d.Actions[firstByte] = createOpenTagActionSingle(name, firstByte)
	// }

	return nil
}

// createCloseTagActionSingle creates an [Action] for an interpretation of a single, 1-byte long closing tag.
// It returns a [Token] with type [TokenClosingTag], if the trigger symbol is not the
// first in the input string, and a token with type [TokenText], along with adding a [Warning] otherwise.
func createCloseTagActionSingle(name string, openTagID byte, char byte) Action {
	return func(input string, i int, warns *[]Warning) (token Token, stride int) {
		// string representation of the tag
		tag := input[i : i+1]

		// default happy params
		t := TokenClosingTag
		inner := ""

		// if the trigger symbol is at the very beginning of the input, return it as token with type [TokenText]
		// and add a Warning
		if i == 0 {
			t = TokenText
			inner = tag

			desc := "Unescaped closing tag with name '" + name + "' found at the very beginning of the input."

			*warns = append(*warns, Warning{
				Issue:       IssueMisplacedClosingTag,
				Pos:         i,
				Description: desc,
			})
		}

		// otherwise return proper closing tag token
		token = Token{
			Name:         name,
			Type:         t,
			TagID:        char,
			OpeningTagID: openTagID,
			Pos:          i,
			Width:        1,
			Raw:          tag,
			Inner:        inner,
		}

		// in any case we process exactly 1 byte
		stride = 1
		return
	}
}

// createCloseTagActionMultiple creates new closing tag with the provided name and opening tag ID as openTagID,
// which starts with sequence of bytes - seq.
func createCloseTagActionMultiple(name string, openTagID byte, seq []byte) Action {
	return func(input string, i int, warns *[]Warning) (token Token, stride int) {
		// 1. Checking the closing tag consistency

		// TODO: finish it
		// 1.1 If the tag is placed at the very beginning of the string, return token with type [TokenText]
		// and add a [Warning]
		if i == 0 {
			token = Token{
				Name:         name,
				Type:         TokenText,
				TagID:        seq[0],
				OpeningTagID: openTagID,
				Pos:          i,
				// Width: ,
			}
		}

		return
	}
}
