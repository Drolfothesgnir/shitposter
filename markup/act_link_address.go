package markup

import (
	"strconv"
	"unicode/utf8"
)

// actLinkAddress checks if there is a symbol sequence forming an URL pattern after the SymbolLinkTextEnd.
// It is triggered by an occurance of the SymbolLinkTextEnd symbol.
//
// # Behaviour:
//
//  1. If the next symbol after the SymbolLinkTextEnd is not a SymbolLinkURLStart or is the EOL,
//     then the inital SymbolLinkTextEnd is treated as plain text and the corresponding Token and a Warning will be
//     returned.
//
//  2. In case of URL opening tag is present and well-placed but the closing tag is not found through
//     the rest of the string, the initial sequence of text closing and URL opening tags will be
//     treated as plain text and the corresponding token and a Warning will be returned.
//
//  3. If all special symbol are placed correctly but the URL closing tag is placed immediately after
//     the opening tag and, therefore the URL is empty, token TypeURL will be returned with value
//     of the special symbols sequence of SymbolLinkTextEnd, SymbolLinkURLStart and SymbolLinkURLEnd
//     and the corresponding Warning will be added.
//
// # Example:
//
// in string "abc](https://google.com)", the part "](https://google.com)" will be considered
// a valid URL pattern.
func actLinkAddress(input string, i int, warns *[]Warning) (token Token, stride int) {
	n := len(input)

	// happy path first
	// if the SymbolLinkTextEnd is not the last one in the string and is followed by SymbolLinkURLStart and
	// the sequence is eventually terminated with SymbolLinkURLEnd, return token TypeURL
	if i+1 < n && Symbol(input[i+1]) == SymbolLinkURLStart {
		// index of the next character after the URL starting tag
		contentStartIdx := i + 2

		// index of the URL closing tag
		closingTagIdx := -1

		// searching for the closing symbol
		for idx := contentStartIdx; idx < n; idx++ {
			if Symbol(input[idx]) == SymbolLinkURLEnd {
				closingTagIdx = idx
				break
			}
		}

		// length of the string content between the URL opening and closing tags
		contentLen := closingTagIdx - contentStartIdx

		// if the closing tag is found and the URL sequence is not empty, return the entire sequence,
		// starting from the SymbolLinkTextEnd up to and including SymbolLinkURLEnd, as a token TypeUrl
		if contentLen > 0 {
			token = Token{
				Type: TypeLinkAddress,
				Pos:  i,
				Len:  closingTagIdx + 1 - i,
				Val:  input[i : closingTagIdx+1],
			}

			stride = closingTagIdx + 1 - i
			return
		}

		// if the closing tag is found but there are no characters between the URL open and closing tags,
		// return token TypeURL with effectively empty URL and add a Warning of empty url
		if contentLen == 0 {
			token = Token{
				Type: TypeLinkAddress,
				Pos:  i,
				// 1 byte for SymbolLinkTextEnd,
				// 1 byte for SymbolLinkURLStart and
				// 1 byte for SymbolLinkURLEnd
				Len: 3,
				Val: input[i : i+3],
			}

			desc := "Empty URL string at index " + strconv.Itoa(i+1) + "."

			*warns = append(*warns, Warning{
				Node:        NodeLink,
				Index:       i + 1,
				Near:        input[i : i+3],
				Issue:       IssueMalformedLink,
				Description: desc,
			})

			stride = 3
			return
		}

		// if the closing tag was not found in the rest of the string, return SymbolLinkTextEnd and
		// SymbolLinkURLStart sequence as a token TypeText and add a Warning of unexpected
		if closingTagIdx == -1 {
			token = Token{
				Type: TypeText,
				Pos:  i,
				Len:  2,
				Val:  input[i : i+2],
			}

			desc := "Unexpected end of the line: expected to find '" +
				string(SymbolLinkTextEnd) + "' after the index " + strconv.Itoa(i+1) +
				" but the rest of the string doesn't contain it."

			*warns = append(*warns, Warning{
				Node:        NodeText,
				Index:       n,
				Issue:       IssueUnexpectedEOL,
				Description: desc,
			})

			stride = 2
			return
		}
	}

	// case of the last symbol
	// if the SymbolLinkTextEnd is the last in the string, return it as token TypeText
	// and add a Warning of unexpected EOL
	if i+1 == n {
		token = Token{
			Type: TypeText,
			Pos:  i,
			Len:  1,
			Val:  input[i : i+1],
		}

		desc := "Unexpected end of the line: expected to find '" +
			string(SymbolLinkURLStart) + "', got EOL."

		*warns = append(*warns, Warning{
			Node:        NodeText,
			Index:       n,
			Issue:       IssueUnexpectedEOL,
			Description: desc,
		})

		stride = 1
		return
	}

	// final case: unexpected next symbol

	// width in bytes of the next symbol
	width := 1

	// if the next symbol is a multi-byte char, decode it
	if input[i+1] >= 128 {
		_, width = utf8.DecodeRuneInString(input[i+1:])
	}

	// return Token TypeText and add a Warning of an unexpected symbol
	token = Token{
		Type: TypeText,
		Pos:  i,
		Len:  1,
		Val:  input[i : i+1],
	}

	desc := "Unexpected symbol: expected to find '" +
		string(SymbolLinkURLStart) + "', got '" + input[i+1:i+1+width] + "'."

	*warns = append(*warns, Warning{
		Node:        NodeText,
		Index:       i + 1,
		Near:        input[i : i+1+width],
		Issue:       IssueUnexpectedSymbol,
		Description: desc,
	})

	stride = 1

	return
}
