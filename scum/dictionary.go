// # Shitposter's Completely User-customizable Markup.
//
// It is expected to be used to define INLINE markup for the post content.
// You can set opening, closing, universal and greedy tags along with the escape symbol dynamically during runtime.
// Tags will have the tag name of your choice and the end AST will be built based on it.
//
// WARNING: It works exclusively with simple 1-byte long ASCII symbols as tags.
//
// # Notes and Policies.
//
//  1. This is My Toy and it was created for fun.
//  2. You can use only 1-byte long ASCII symbols for tags.
//  3. A tag can consist of at most [MaxTagLength] characters.
//  4. Nested tags with the same ID will have no effect. Children of the repeated descendants will become
//     children of the "oldest" original tag and the duplicates will not end up in the final AST.
//  5. A universal Tag is one, which has both the opening and the closing tags the same.
//  6. Escape tag ID will be reserved, likely to some ASCII control character.
//  7. The parser will try to make sense out of the User's gibberish and will not return any errors but only a slice of [Warning].
//
// # Behaviour. This will likely change in the future.
//
// Properties of Tags:
//
//  1. ID. Each Tag has unique byte, which triggers the tag's corresponding Action and starts the process of Tokenization.
//     It also serves as unique ID of the Tag and is used for fast lookup for the Tag's info.
//
//  2. Name. Each Tag has Name string associated with it. It does not need to be unique. It's used during the parsing process
//     for naming the AST's Node.
//
//  3. Greed. If a Tag has Greed level > 0, it becomes "greedy". Each Tag can have 3 Greed levels:
//
//     3.1. 0 Level, default Greed level. During the tokenization process, only Tag's string representation will be considered as a Token
//     with the corresponding ID, all next characters will be tokenized normally. Example: Imagine a Tag with ID '$',
//     name "BOLD", string representation "$$" and Greed 0. In string "$$hello$$", the first token will be with ID '$' and have value "$$".
//
//     3.2. 1 Level. All bytes starting from the opening tag and including the closing tag will be considered as this token's value. If
//     the closing tag is not present during the tokenization, the text token will be returned and it's value will
//     be the opening tag string representation only. All next characters will be tokenized normally.
//
//     3.2.1. Example 1: Imagine a tag with id '(', name "URL", tag string "(" and Greed level 1. Imagine also its closing tag with the same
//     name, ID ')' and the tag string ")". In the string "[**some link**](https://google.com)", the whole part "(https://google.com)" will
//     be considered a token and its internal value will be https://google.com.
//
//     3.2.2. Example 2: Imagine the same setting string as in the first example, but the string is now "[**some link**](https://google.com",
//     that is, now closing tag for the "URL". In this case the "(" part will be a text token, and all next bytes parsed normally.
//
//     3.3 2 Level. Same logic as with Level 1, but in the case of missing closing tag, the rest of the string will still be consumed, and
//     token will be for tag not a text. Example: The same setting as in the Example 2, 3.2.2. - "[**some link**](https://google.com". The
//     substring, starting from "(" and to the end, will be a token and it's internal value will be "https://google.com".
//
//  4. Sequence. You can construct your tags from at most [MaxTagLength] (4 by default). You can create tag like this "$}{|". The Sequence
//     is a slice of bytes with length of the defined tag, and with indexes corresponding to indexes of chars in the tag. For tag "$}{|"
//     Sequence will be []byte{'$', '}', '{', '|'}.
//
//  5. Rule. Each UNIVERSAL SINGLE BYTE TAG can have 3 different Rules available to it:
//
//     5.1 0 Rule - No Rule. Default Rule value. Does nothing.
//
//     5.2 1 - Intra-word Rule. Only available for the single-byte "NON-GREEDY" tags. If char, which normally defines a 1-byte long Tag, has alphanumerics, punctuation
//     symbols, OR THE SAME TAG SYMBOL, on BOTH sides, the it will be considered a plain text and not a Tag trigger.
//
//     5.2.1.	Example 1: '_' defines a Tag with name "UNDERLINE" and has
//     Rule 1 on it. In string "image_from_.png" both '_' will be considered a plain text.
//
//     5.2.2. Example 2: In string "_image_from_net.jpg", only 2 last '_'
//     symbols will be a plain text. The first '_' will trigger the Action for the "UNDERLINE" tag, because it has nothing on the left.
//
//     5.2.3. Example 3:  In string "_hello__", both the first and the last '_' will be considered a tokens, but the one before
//     the last will not, since it has "o" on the left and "_" on the right.
//
//     5.3 2 Rule - Tag-VS-Content Rule. Only available for single-byte GREEDY tags. When you have single char tag, like '`',
//     and your text contains symbol "`", it will be interpreted as a closing tag. This might be a problem. Consider Example 1: '`' defines a greedy universal tag
//     with name "CODE". In the string "here is some code: `const rawStr = `hello world`;`". In this case there will be tokens: type text with value ("here is some code: "),
//     type tag with name "CODE" and value "`const rawStr = `", type text with value "hello world" and type tag with name "CODE" and value "`;`". It's likely
//     not what the User intended. The Tag-VS-Content Rule solves this problem by imposing two conditions: 1) You can repeat symbol in tags how, but the lengths of
//     the opening and closing tags must be the same. 2) Length of tags must differ ftom the length of the symbol sequence in the plain text.
//
//     5.3.1. Example 2: The setting from the example 1, but now we make tag length equal 3, by making each tag "```":
//     "here is some code: ```const rawStr = `hello world`;```". Now the "CODE" tag will capture entire "```const rawStr = `hello world`;```" part.
//
//  6. Opening/Closing tag IDs. Each Tag has OpenID and ClosID fields. You have to set at least one of them to ensure the Parser will process them correctly. To make
//     a Tag an opening tag, you have to set its ClosID value to something other than '\000', to inform the Parser that this tag has to be closed with some other Tag.
//     The same with the closing tags: just set the OpenID value to something. To make a Tag universal you have to set both values to the Tag's ID. You can also
//     create one-to-many relationships between tags, by setting Open/ClosID to some conventional or reserved byte, like a control character from ASCII.
//
//     6.1. Example 1 - The Tag expected to be closed with specific other Tag: Imagine you've defined the non-greedy single-byte Tag with name "LINK_TEXT_START",
//     ID '[' and ClosID ']'. Then you've defined the non-greedy Tag with name "LINK_TEXT_END", ID ']' and OpenID '['. During the parsing of the tokens, the Parser,
//     when first encounters the "LINK_TEXT_START", saves its ID to the stack. While the '[' is at the top of the stack and the Parser encounters a closing Tag
//     it checks if the encountered Tag ID is equal to the "LINK_TEXT_START's" ClosID. What will happen next is a good question.
//
//     6.2 Example 2 - one-to-many Tag relationships: Imagine you want to have link URLs and image URLs in your mark up. You create a single-byte Tag with
//     ID '[', name "LINK_URL_START" and ClosID ']'. Then you create a double-byte Tag with sequence "![", ID '!', name "IMAGE_URL_START" and ClosID again ']'.
//     Then you create a single-byte Tag with ID ']', name "LINK_TEXT_END" and OpenID of some reserved or non-printable ASCII character, whatever except '\000'.
//     Now "LINK_TEXT_END" is compatible with both Tags defined before and can close any of them.
//
//  7. GreedyChild. You can set your Tag's field GreedyChild to the ID of some greedy Tag. When the Parser encounters the properly closed Tag with set
//     GreedyChild field, if the next Tag is a greedy tag with ID equal to the GreedyChild field of the first Tag, the value of latter will be considered
//     the property of the first tag and will be assigned to the Node of the first Tag.
//
//     7.1 Example: Imagine that you've defined the non-greedy single-byte Tag with ID '[', name "LINK_TEXT_START", ClosID ']', and GreedyChild
//     set to '('. You've also defined the closing single Tag for the first one: ID ']', name "LINK_TEXT_END", OpenID '['. Lastly, you've defined
//     the single-byte greedy Tag with ID '(' and name "LINK_URL_START". You have the input string "[Hello World!](https://hello-world.com)". The Parser
//     will first create the Node for the link text with inner text "Hello World!". Then it will encounter the "(https://hello-world.com)" part, which
//     it will interpret as a greed-consumed value of the "LINK_TEXT_START" and assign it to the created Node.
//
// TODO: add preallocated tag strings inside the dictionary
// TODO: add docs for cases when closing tag does not match the opening and is returned as plain text along with a Warning
// TODO: add docs for escaping
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

// Tag contains all the info about a particular tag, relevant for the tokenizing and parsing.
type Tag struct {
	ID          byte
	Name        string
	Greed       uint8
	Seq         []byte
	Rule        uint8
	OpenID      byte
	ClosID      byte
	GreedyChild byte
}

// Action is a function triggered by a special symbol defined in the [Dictionary].
// It processes the input string strating from the index i and returns a [Token] and,
// possibly, adds a [Warning].
type Action func(input string, i int, warns *[]Warning) (token Token, stride int)

// Dictionary manages creation and deletion of Tags and their corresponding Actions.
type Dictionary struct {
	Actions [256]Action
	Tags    [256]Tag
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

func checkMultiByteTagConsistency(name string, seq []byte, tokenType TokenType, input string, i int, warns *[]Warning) (token Token, stride int, ok bool) {
	n := len(input)
	// 1. Checking the tag consistency

	// 1.1 Checking if the input after index i is similar to the rest of the tag byte sequence
	// we start comparing from i+1 because byte at index i is the same, since it was an Action trigger
	diffIndex, substrShorter := checkByteDifference(input[i+1:], seq[1:])

	// if the input's tag sequence is different from seq, that is diffIndex > -1, return [Token] with
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

		wrong, _, valid := extractNextRune(input[absDiffIndex:])

		quoted := strconv.QuoteRune(wrong)

		got := ""

		if !valid {
			got = "unrecognizable character "
		}

		tagDesc := " "

		switch tokenType {
		case TokenOpeningTag:
			tagDesc = " opening "
		case TokenClosingTag:
			tagDesc = " closing "
		case TokenUniversalTag:
			tagDesc = " universal "
		case TokenGreedyTag:
			tagDesc = " greedy "
		}

		desc := "Unexpected symbol at index " +
			strconv.Itoa(absDiffIndex) +
			" while interpreting the" + tagDesc + "tag with name '" +
			name + "': expected to get '" + string(seq[diffIndex+1]) +
			"', got " + got + quoted + "."

		*warns = append(*warns, Warning{
			Issue:       IssueUnexpectedSymbol,
			Pos:         absDiffIndex,
			Description: desc,
		})

		// we've processed all bytes from the index i and up to index of the first divergence
		stride = absDiffIndex - i

		ok = true
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

		ok = true
		return
	}

	return
}

// createOpenTagMultiple creates an Action for a new opening tag with the provided name, which starts with sequence of bytes - seq.
func createOpenTagActionMultiple(name string, seq []byte) Action {
	return func(input string, i int, warns *[]Warning) (token Token, stride int) {
		// 1. Checking the opening tag consistency
		t, s, tagInconsistent := checkMultiByteTagConsistency(name, seq, TokenOpeningTag, input, i, warns)

		if tagInconsistent {
			token = t
			stride = s
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

		// we've processed entire opening tag sequence
		stride = seqLen

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

	if l > 1 {
		d.Actions[firstByte] = createCloseTagActionMultiple(name, openTagID, closeSeq)
	} else {
		d.Actions[firstByte] = createCloseTagActionSingle(name, openTagID, firstByte)
	}

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
		t, s, tagInconsistent := checkMultiByteTagConsistency(name, seq, TokenClosingTag, input, i, warns)

		if tagInconsistent {
			token = t
			stride = s
			return
		}

		tokenType := TokenClosingTag
		inner := ""

		seqLen := len(seq)
		tag := input[i : i+seqLen]

		// 2. Checking if the closing tag is the very beginning of the string

		// if the string starts with the closing tag sequence return token with type [TokenText] and add a [Warning]
		if i == 0 {
			tokenType = TokenText
			inner = tag

			desc := "Closing tag with name '" + name + "' found at the very beginning of the input."

			*warns = append(*warns, Warning{
				Issue:       IssueMisplacedClosingTag,
				Pos:         i,
				Description: desc,
			})
		}

		// 3. Happy case
		token = Token{
			Name:         name,
			Type:         tokenType,
			TagID:        seq[0],
			OpeningTagID: openTagID,
			Pos:          i,
			Width:        seqLen,
			Raw:          tag,
			Inner:        inner,
		}

		// we've processed only the closing byte sequence at the beginning of the string
		stride = seqLen
		return
	}
}
