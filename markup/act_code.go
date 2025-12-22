package markup

import (
	"strconv"
	"strings"
)

// actCode can return three different token types, [TypeCodeInline], [TypeCodeBlock] or [TypeText] and a possible
// [Warning].
//
// Behaviour:
//
// actCode adheres to the N+1 rule of code opening tag, that is if the sequence of a [SymbolCode] is used somewhere in the
// actual code block, then lengths of the opening and closing [SymbolCode] sequences must be at least one char longer or shorter,
// to not confuse the code block sequence with the closing tag.
//
// When proper opening and closing tags present, if length of the tags are more than or equal to 3, then
// the code sequence will be considered a block and a token [TypeCodeBlock] will be returned.
//
// Otherwise, a token [TypeCodeInline] will be returned.
//
// If the trigger symbol is the last in the string, actCode will return token [TypeText] and a
// [Warning] with an unexpected EOL issue.
//
// If the rest of the input string consists only of [SymbolCode] chars, the function will treat the entire input as
// a plain text and return token [TypeText] along with adding a [Warning].
//
// If the opening tag sequence consists of K symbols, the closing sequence must be exactly K symbols long.
// If the closing sequence with width of K is not found through the rest of the string, there are two outcomes:
//  1. The opening sequence is shorter than 3 symbols, in which case the opening sequence is treated as a plain text
//     and a token [TypeText] with value of only opening sequence is returned along with a [Warning] with an unclosed tag issue.
//  2. The opening sequence has length more than or equal 3, in which case the entire string starting from the index of the
//     trigger symbol is trated as a code block and a token [TypeCodeBlock] is returned along with a [Warning] with an
//     unclosed tag issue.
func actCode(input string, i int, warns *[]Warning) (token Token, stride int) {
	n := len(input)

	// 1. Checking the last char case.
	// if the code symbol is the last in the string return it as a text token and add a Warning
	if i+1 == n {
		token = Token{
			Type: TypeText,
			Pos:  i,
			Len:  1,
			Val:  input[i:],
		}

		*warns = append(*warns, Warning{
			Node:        NodeText,
			Index:       i + 1,
			Issue:       IssueUnexpectedEOL,
			Description: "Unexpected end of the line: expected to get a character, got EOL instead.",
		})

		// explicitely signal the main loop that we have proccessed only the original symbol.
		stride = 1

		return
	}

	// 2. Calculating the opening tag length

	// calculating length of the longest SymbolCode substring as a opening tag

	// account for the inital character
	openSeqLen := 1

	for idx := i + 1; idx < n && Symbol(input[idx]) == SymbolCode; idx++ {
		openSeqLen++
	}

	// index of the next byte in the input
	contentStartIdx := i + openSeqLen

	// 3. Checking the only-code symbols case

	// if the length of the opening tag equals to the length of the rest of the string,
	// return token TypeText and add a Warning
	if openSeqLen == n-i {
		token = Token{
			Type: TypeText,
			Pos:  i,
			Len:  openSeqLen,
			Val:  input[i : i+openSeqLen],
		}

		desc := "Malformed code sequence: the input consists only of '" +
			input[i:i+1] + "', starting from the index " + strconv.Itoa(i)

		*warns = append(*warns, Warning{
			Node:        NodeText,
			Index:       i,
			Description: desc,
			Issue:       IssueMalformedCodeSequence,
		})

		stride = openSeqLen
		return
	}

	// 3. Searching for the closing tag
	// NOTE: all this effort was put only for incorporating [strings.IndexByte] into the search.

	// next we search for the closing tag and consider all SymbolCode
	// sequences, which lengths are not equal to the starting tag length,
	// a part of the code block, according to the N+1 tag rule.

	// IMPORTANT:
	// The outline:
	// 		1. We look for the next code symbol with [strings.IndexByte]. If no symbol found,
	//       tag unclosed, stop searching.
	// 	  2. Go to the found symbol's index and start counting code symbols from there.
	//    3. If this sequence has same length as the opening one, we've found our closing tag, stop searching.
	//       Else go to 1.

	// count of SymbolCode chars in the possible closing sequence
	closeSeqLen := 0

	// index of the first code symbol in the closing tag
	closeSeqStartIdx := -1

	// we start looking the next code symbol from here
	searchStartIdx := contentStartIdx + 1

	// we will loop until there are no symbols left to check
	for searchStartIdx < n {

		// relative index of the first SymbolCode in the rest of the string,
		// it cannot be used as input string index and must be offsetted with the search start position
		nextCodeSymIdx := strings.IndexByte(input[searchStartIdx:], byte(SymbolCode))

		// if there are no code symbols in the rest of the string it means the tag is unclosed
		// and we have to stop looking
		if nextCodeSymIdx == -1 {
			break
		} else {
			// else making next code symbol index absolute and usable for the original input indexing
			nextCodeSymIdx += searchStartIdx
		}

		// assigning possible closing sequence start index
		closeSeqStartIdx = nextCodeSymIdx

		// else, if we found the next code symbol, we count the number of the consecutive symbols
		closeSeqLen = 1

		// counting the actual sequence length
		for idx := nextCodeSymIdx + 1; idx < n && Symbol(input[idx]) == SymbolCode; idx++ {
			closeSeqLen++
		}

		// after counting we first check if the recent sequence has the same length as the opening one
		if closeSeqLen == openSeqLen {
			// if yes, then we've found our closing tag, and we stop the search
			break
		}

		// moving the search start index to the next character after the current plain text
		searchStartIdx = nextCodeSymIdx + closeSeqLen + 1

		// if no, we reset the closing sequence start pointer and length and continue the search
		closeSeqLen = 0
		closeSeqStartIdx = -1
	}

	// 4. Checking the missing closing tag case

	// if the code block is considered unclosed, that is, we haven't found SymbolCode sequence
	// with the same length as the starting one before the end of the string, we:
	//   - we return tag as plain text and continue tokenize the rest of the string
	//   - or save the substring as a code block if the starting tag has length >= 3, that is, opens a code block
	if closeSeqStartIdx == -1 {

		var nodeType NodeType

		// return the opening tag as a plain text if the tag is inline
		if openSeqLen < 3 {
			token = Token{
				Type: TypeText,
				Pos:  i,
				Len:  contentStartIdx - i,
				Val:  input[i:contentStartIdx],
			}

			nodeType = NodeText

			// signaling the main loop that we have processed only the opening tag
			stride = contentStartIdx - i

			// returning unclosed CodeBlock
		} else {
			token = Token{
				Type: TypeCodeBlock,
				Pos:  i,
				Len:  n - i,
				Val:  input[i:],
			}

			nodeType = NodeCode

			// signaling to the main loop that we've proccessed all the remaining bytes
			stride = n - i
		}

		*warns = append(*warns, Warning{
			Node:        nodeType,
			Index:       i,
			Issue:       IssueUnclosedTag,
			Description: "Unclosed code block at index " + strconv.Itoa(i),
		})

		return
	}

	// 6. Happy case

	// when code block is closed, we choose Type based on the length of the opening sequence
	var t Type

	if openSeqLen >= 3 {
		t = TypeCodeBlock
	} else {
		t = TypeCodeInline
	}

	// assuming all code symbols are the same we will calculate the end of the closing tag by
	// simply multiplying the width of the SymbolCode by the count of symbols in the tag +
	// the starting index of the tag
	codeEnd := closeSeqStartIdx + closeSeqLen

	token = Token{
		Type: t,
		Pos:  i,
		Len:  codeEnd - i,
		Val:  input[i:codeEnd],
	}

	// adjusting main loop pointer offset
	stride = codeEnd - i
	return
}
