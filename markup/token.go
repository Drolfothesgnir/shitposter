package markup

import "unicode/utf8"

// Token represents a single markdown token.
type Token struct {
	Type Type // Token type: bold, italic, text, etc.
	Pos  int  // Starting byte position of the token in the original markdown string.

	// Len is the length of the token's byte sequence.
	//
	// WARNING: This is not the visible text length.
	//
	// It is the byte length of the underlying string as used internally by the tokenizer.
	// Do not use this field for visible text length calculations; use rune counting instead.
	Len int

	// Val is the exact token string:
	// - for tags: the literal tag/delimiter ("**", "*", "[", etc.),
	// - for text: the raw text content.
	Val string
}

// Type defines the kind of token, e.g. bold, italic, link start, etc.
type Type int

const (
	TypeBold Type = iota
	TypeItalic
	TypeStrikethrough
	TypeCodeBlock
	TypeCodeInline
	TypeEscapeSequence
	TypeUnderline
	TypeLinkTextStart
	TypeLinkAddress
	TypePlainURL
	TypeImageTextStart
	TypeText
)

// Tag defines the string representation of markdown tags,
// e.g. "**" for bold, "*" / "_" for italic.

const (
	TagBold          = "**"
	TagStrikethrough = "~~"
	TagItalic        = "*"
	TagCode          = "`"
	TagLinkTextStart = "["
	TagLinkTextEnd   = "]"
	TagLinkURLStart  = "("
	TagLinkURLEnd    = ")"
	TagImageMarker   = "!"
	TagEscape        = "\\"
)

type Symbol byte

const (
	SymbolStrikethrough Symbol = '~'
	SymbolEscape        Symbol = '\\'
	SymbolCode          Symbol = '`'
	SymbolItalic        Symbol = '*'
	SymbolUnderline     Symbol = '_'
	SymbolImageMarker   Symbol = '!'
	SymbolLinkTextStart Symbol = '['
	SymbolLinkTextEnd   Symbol = ']'
	SymbolLinkURLStart  Symbol = '('
	SymbolLinkURLEnd    Symbol = ')'
)

// action defines a function which accepts an input string, an index of a special character in it
// and a pointer to the Warning slice, to process it and return a token and a number of
// processed bytes.
type action func(input string, idx int, warns *[]Warning) (token Token, stride int)

// symToAction maps special characters to their corresponding actions, also effectively serving as
// a way to check if the byte is special
//
// WARNING: The tokenizer works with ONLY 1-byte ASCII characters as special symbols.
// Using multi-byte special symbols will cause unexpected behaviour.
var symToAction [256]action

// init helps to assign actions to their corresponding special symbols.
//
// Actions cannot be assigned in the literal above because they are using symToAction
// while being part of it, which causes a circular dependecy and makes compiler throw an
// error.
func init() {
	symToAction[SymbolCode] = actCode
	symToAction[SymbolEscape] = actEscape
	symToAction[SymbolItalic] = actBoldOrItalic
	symToAction[SymbolStrikethrough] = actStrikethrough
	symToAction[SymbolUnderline] = actUnderline
	symToAction[SymbolImageMarker] = actImageMarker
	symToAction[SymbolLinkTextStart] = actLinkTextStart
	symToAction[SymbolLinkTextEnd] = actLinkAddress
}

// TODO: account for [utf8.RuneError]
// Tokenize processes the input string rune-wise and outputs a slice of Tokens and a slice of Warnings.
func Tokenize(input string) (tokens []Token, warnings []Warning) {

	// guessing the token number to minimize the number of the slice resizes
	tokens = make([]Token, 0, len(input)/4)

	n := len(input)

	// starting index of the plain text sequence
	textStart := 0

	prevRune := '\000'

	for i := 0; i < n; {

		// current byte
		b := input[i]

		act := symToAction[b]

		// isRealTag is true if the current symbol is either a SymbolUnderline considered
		// special, not a plain text, or other special symbol
		isRealTag := false

		// because of intra-word rule for underscores, the occurance of the SymbolUnderline
		// is a special case and is handled explicitely in the main loop
		if act != nil {
			if Symbol(b) == SymbolUnderline {
				// applying action only if the Underline is a tag
				isRealTag = isUnderlineTag(input, i, n, prevRune)
			} else {
				isRealTag = true
			}
		}

		// if we've encountered real special symbol,
		// we flushing the text and and performing action
		if isRealTag {

			// only if text is not empty
			if textStart < i {

				text := Token{
					Type: TypeText,
					Pos:  textStart,
					Len:  i - textStart,
					Val:  input[textStart:i],
				}

				tokens = append(tokens, text)
			}

			token, stride := act(input, i, &warnings)

			tokens = append(tokens, token)

			// skipping the bytes processed by the action
			i += stride

			// resetting text start pointer
			textStart = i

			// resetting the previous character
			// FIXME: don't reset to the null, but to the previous plain text
			prevRune = '\000'

			continue
		}

		// else the symbol is a plain text

		// if the value of the first byte is less than 128, then it's a simple ASCII char and
		// has width of 1 byte and the prev rune becomes the byte itself
		if b < 128 {
			i += 1
			prevRune = rune(b)
		} else {
			// else the char must be multi-byte symbol and we have to decode it
			r, w := utf8.DecodeRuneInString(input[i:])

			i += w
			prevRune = r
		}
	}

	// final text flushing
	if textStart < n {
		token := Token{
			Type: TypeText,
			Pos:  textStart,
			Len:  n - textStart,
			Val:  input[textStart:],
		}

		tokens = append(tokens, token)
	}

	return
}
