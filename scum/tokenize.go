package scum

// Tokenize transforms the input string into the sequence of Tokens.
// It can emit Warnings during the process.
// NOTE: i will likely reuse the warns slice during the parsing, so it's passed as reference instead of being created here.
func Tokenize(d *Dictionary, input string, warns *Warnings) (tokens []Token) {
	n := len(input)

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

		token, stride, skip := act(d, b, input, i, warns)

		// stride = max(stride, 1)

		// if the current byte is special but for some reason is not being part of some Tag
		// we consider it as a plain text and move on
		if skip {
			i += stride
			continue
		}

		// if the current byte is part of a Tag, we first trying to flush existing plain text as a Token
		textLen := i - textStart

		// but only if the text string is not empty
		if textLen > 0 {
			tokens = append(tokens, Token{
				Type:    TokenText,
				Pos:     textStart,
				Width:   textLen,
				Raw:     Span{textStart, i},
				Payload: Span{textStart, i},
			})
		}

		// adjusting the loop pointer to account the processed bytes
		i += stride

		// resetting the text starting point to the index after the current special symbol's byte sequence instead of i, because
		// this symbol is already considered special
		textStart = i

		// appending the Action's Token
		tokens = append(tokens, token)
	}

	// if there are no special symbols from the last text string's start to the very end of the input
	// we flush the text as a Token
	if textStart < n {
		tokens = append(tokens, Token{
			Type:    TokenText,
			Pos:     textStart,
			Width:   n - textStart,
			Raw:     Span{textStart, n},
			Payload: Span{textStart, n},
		})
	}

	return
}
