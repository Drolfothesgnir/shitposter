package scum

// TokenizerOutput is the result of the tokenization process.
type TokenizerOutput struct {
	// Tokens is the sequence of Tokens.
	Tokens []Token
	// TextLen is the total length of the text in the input string.
	TextLen int
	// TagsTotal is the total count of Tags in the input string.
	TagsTotal int
	// UniversalTags is the total count of universal Tags in the input string, not accounting for greedy Tags.
	UniversalTags int
	// OpenTags is the total count of opening Tags in the input string, not accounting for greedy Tags.
	OpenTags int
	// CloseTags is the total count of closing Tags in the input string, not accounting for greedy Tags.
	CloseTags int
	// TextTokens is the total count of text Tokens in the input string.
	TextTokens int
	// Attributes is the total count of Attributes in the input string.
	Attributes int
}

// TokenizerState helps gather metadata about the parsed input string.
type TokenizerState struct {
	TagsTotal     int
	UniversalTags int
	OpenTags      int
	CloseTags     int
	Attributes    int
}

// Tokenize transforms the input string into the sequence of Tokens.
// It can emit Warnings during the process.
// NOTE: i will likely reuse the warns slice during the parsing, so it's passed as reference instead of being created here.
func Tokenize(d *Dictionary, input string, warns *Warnings) (out TokenizerOutput) {
	n := len(input)

	var s TokenizerState

	// the place where current plain string started
	textStart := 0

	for i := 0; i < n; {

		b := input[i]

		act, isSpecial := d.Action(b)

		// if the current byte is not a special one, we simply move forward withe the loop
		if !isSpecial {
			i++
			continue
		}

		token, stride, skip := act(d, &s, warns, input, b, i)

		// stride = max(stride, 1)

		// if the current byte is special but for some reason is not being part of some Tag
		// we consider it as a plain text and move on
		if skip {
			i += stride
			continue
		}

		// if the current byte is part of a Tag, we first try to flush existing plain text as a Token
		textLen := i - textStart

		// but only if the text string is not empty
		if textLen > 0 {
			out.Tokens = append(out.Tokens, Token{
				Type:    TokenText,
				Pos:     textStart,
				Width:   textLen,
				Payload: Span{textStart, i},
			})

			out.TextTokens++
			out.TextLen += textLen
		}

		// adjusting the loop pointer to account the processed bytes
		i += stride

		// resetting the text starting point to the index after the current special symbol's byte sequence instead of i, because
		// this symbol is already considered special
		textStart = i

		// appending the Action's Token
		out.Tokens = append(out.Tokens, token)
	}

	// if there are no special symbols from the last text string's start to the very end of the input
	// we flush the text as a Token
	if textStart < n {
		out.Tokens = append(out.Tokens, Token{
			Type:    TokenText,
			Pos:     textStart,
			Width:   n - textStart,
			Payload: Span{textStart, n},
		})

		out.TextTokens++
		out.TextLen += n - textStart
	}

	out.TagsTotal = s.TagsTotal
	out.UniversalTags = s.UniversalTags
	out.OpenTags = s.OpenTags
	out.CloseTags = s.CloseTags
	out.Attributes = s.Attributes

	return
}
