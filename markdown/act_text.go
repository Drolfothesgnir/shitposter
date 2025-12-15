package markdown

// actText finds longest plain text substring in the substr and returns corresponding TypeText Token.
func actText(substr string, cur rune, symLen int, i int, isLastRune bool) (token Token, warnings []Warning, stride int, ok bool) {

	// since actText returns token anyway we define 'ok' as true immediately.
	ok = true

	// textEnd is index of either the first special symbol occurance or the EOL,
	// if the string doesn't contain any special symbols.
	textEnd := len(substr)

	// looking for the first special symbol in the string
	for idx, r := range substr {
		if isSpecialSymbol(r) {
			textEnd = idx
			break
		}
	}

	// otherwise the textEnd remains EOL.

	seq := substr[:textEnd]

	nBytes := len(seq)

	token = Token{
		Type: TypeText,
		Pos:  i,
		Len:  nBytes,
		Val:  seq,
	}

	// signalling the main loop how many bytes we've processed
	stride = nBytes
	return
}
