package markdown

import "strconv"

// ...
func actPlainURL(input string, i int, warns *[]Warning) (token Token, stride int) {

	n := len(input)

	// happy path first
	// if the encountered [SymbolLinkURLStart] is not the last in the string, we search for the
	// [SymbolLinkURLEnd]
	if i+1 < n {

		// index of the found [SymbolLinkURLEnd]
		closingTagIdx := -1
		for idx := i + 1; idx < n; idx++ {
			if Symbol(input[idx]) == SymbolLinkURLEnd {
				closingTagIdx = idx
				break
			}
		}

		// len in bytes of the string between the opening and closing tags
		urlLength := closingTagIdx - i - 1

		// if we've found the closing tag, and the URL is not empty, we create the token
		if closingTagIdx > -1 && urlLength > 0 {
			// telling the main loop that we've processed the URL and both tags
			stride = closingTagIdx + 1 - i

			token = Token{
				Type: TypePlainURL,
				Pos:  i,
				Len:  stride,
				Val:  input[i : closingTagIdx+1],
			}
			return
		}

		// else if the URL is empty we still return a token [TypePlainURL], but add a [Warning]
		if urlLength == 0 {
			token = Token{
				Type: TypePlainURL,
				Pos:  i,
				Len:  2, // 1 byte for the each tag
				Val:  input[i : i+2],
			}

			stride = 2

			desc := "Empty URL string at index " + strconv.Itoa(i+1) + "."

			*warns = append(*warns, Warning{
				Node:        NodeLink,
				Index:       i + 1,
				Near:        input[i : i+2],
				Issue:       IssueMalformedLink,
				Description: desc,
			})

			return
		}

		// else if the closing tag was not found, we return a token [TypeText] with value
		// of single opening tag, and add a [Warning], so the rest of the string,
		// after the [SymbolLinkURLStart], is processed as normal
		// as normal
		if closingTagIdx == -1 {
			token = Token{
				Type: TypeText,
				Pos:  i,
				Len:  1,
				Val:  input[i : i+1],
			}

			desc := "Unexpected end of the line: expected to find '" +
				string(SymbolLinkTextEnd) + "' after the index " + strconv.Itoa(i) +
				" but the rest of the string doesn't contain it."

			*warns = append(*warns, Warning{
				Node:        NodeText,
				Index:       n,
				Issue:       IssueUnexpectedEOL,
				Description: desc,
			})

			stride = 1
			return
		}
	}

	// the last case: when the [SymbolLinkURLStart] is the last in the string

	// in this case return token [TypeText] and a [Warning]
	token = Token{
		Type: TypeText,
		Pos:  i,
		Len:  1,
		Val:  input[i : i+1],
	}

	desc := "Unexpected end of the line: expected to find a valid url, got EOL"

	*warns = append(*warns, Warning{
		Node:        NodeText,
		Index:       n,
		Issue:       IssueUnexpectedEOL,
		Description: desc,
	})

	stride = 1
	return
}
