package markdown

import (
	"fmt"
)

// actCode extracts all characters between varible-width SymbolCode sequences as tags.
// It uses N+1 rule based logic to parse the code block.
//
// Example: if your code block has double backticks inside, "“", then opening tags must contain triple backticks, "```",
// to differentiate between the content and the tags.
//
// WARNING: actCode assumes that SymbolCode is 1-byte long ASCII character.
func actCode(input string, i int, warns *[]Warning) (token Token, stride int) {

	n := len(input)

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

	// calculating length of the longest SymbolCode substring as a opening tag

	// account for the inital character
	openTagLen := 1

	// index of the next byte in the input
	contentStartIdx := i + 1

	onlyCodeSymbols := true

	for idx := contentStartIdx; idx < n; idx++ {
		if Symbol(input[idx]) != SymbolCode {
			contentStartIdx = idx
			onlyCodeSymbols = false
			break
		}

		openTagLen++
	}

	// if all symbols after the initial one are SymbolCode, then the loop above will never break and
	// assign correct value to the contentStartIdx, so in this case contentStartIdx will still equal to
	// the width. if this is the case, we set it to the length of the substring
	if onlyCodeSymbols {
		contentStartIdx = n
	}

	// next we search for the closing tag and consider all SymbolCode
	// sequences, which lengths are not equal to the starting tag length,
	// a part of the code block, according to the N+1 tag rule.

	// index of the first code symbol in the possible closing tag
	seqStartIdx := -1

	// count of SymbolCode chars in the possible closing sequence
	seqLen := 0

	for idx := contentStartIdx; idx < n; idx++ {
		// two cases are possible:
		// 1) next byte is the SymbolCode
		if Symbol(input[idx]) == SymbolCode {
			// then:
			// if the sequence is not started, that is the symbol is the first in count,
			// we reset the sequence
			if seqStartIdx == -1 {
				seqStartIdx = idx
			}

			// we also increment sequence length in any sub-case
			seqLen++

			// 2) next byte is plain text and we have the sequence started
		} else if seqStartIdx > -1 {
			// in this case we first check if the sequence has length equal to the starting tag length

			// if yes, then we've found our closing tag and we stop the loop
			if seqLen == openTagLen {
				break

				// else we reset the sequence starting index and length
			} else {
				seqStartIdx = -1
				seqLen = 0
			}
		}
	}

	// if the code block is considered unclosed, that is, we haven't found SymbolCode sequence
	// with the same length as the starting one before the end of the string, we:
	//   - we return tag as plain text and continue tokenize the rest of the string
	//   - or save the substring as a code block if the starting tag has length >= 3, that is, opens a code block

	if seqStartIdx == -1 {
		var nodeType NodeType

		isBlockCode := openTagLen >= 3

		// return the opening tag as a plain text if the tag is inline
		if !isBlockCode {
			token = Token{
				Type: TypeText,
				Pos:  i,
				Len:  contentStartIdx,
				Val:  input[i:contentStartIdx],
			}

			nodeType = NodeText

			// signaling the main loop that we have processed only the opening tag
			stride = contentStartIdx

			// returning unclosed CodeBlock
		} else {
			token = Token{
				Type: TypeCodeBlock,
				Pos:  i,
				Len:  n,
				Val:  input[i:],
			}

			// signaling to the main loop that we've proccessed all the remaining bytes
			stride = n
		}

		*warns = append(*warns, Warning{
			Node:        nodeType,
			Index:       i,
			Issue:       IssueUnclosedTag,
			Description: fmt.Sprintf("Unclosed code block at index %d", i),
		})

		return
	}

	// otherwise, when code block is closed, we choose Type based on the length of the opening sequence

	var t Type

	if openTagLen >= 3 {
		t = TypeCodeBlock
	} else {
		t = TypeCodeInline
	}

	// assuming all code symbols are the same we will calculate the end of the closing tag by
	// simply multiplying the width of the SymbolCode by the count of symbols in the tag +
	// the starting index of the tag
	codeEnd := seqStartIdx + seqLen

	token = Token{
		Type: t,
		Pos:  i,
		Len:  codeEnd,
		Val:  input[i:codeEnd],
	}

	// adjusting main loop pointer offset
	stride = codeEnd
	return
}
