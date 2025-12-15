package markdown

import (
	"fmt"
)

// actCode extracts all characters between varible-width SymbolCode sequences as tags.
// It uses N+1 rule based logic to parse the code block.
//
// Example: if your code block has double backticks inside, "“", then opening tags must contain triple backticks, "```",
// to differentiate between the content and the tags.
func actCode(substr string, cur rune, width int, i int, isLastRune bool) (token Token, warnings []Warning, stride int, ok bool) {

	// actCode returns token in any case so ok = true
	ok = true

	n := len(substr)

	// if the code symbol is the last in the string return it as a text token and add a Warning
	if isLastRune {
		token = Token{
			Type: TypeText,
			Pos:  i,
			Len:  width,
			Val:  substr[:width],
		}

		warnings = []Warning{{
			Node:        NodeText,
			Index:       i + width,
			Issue:       IssueUnexpectedEOL,
			Description: "Unexpected end of the line: expected to get a character, got EOL instead.",
		}}

		// explicitely signal the main loop that we have proccessed only the original symbol.
		stride = width

		return
	}

	// calculating length of the longest SymbolCode substring as a opening tag

	// account for the inital character
	openTagLen := 1

	// index of the next rune in the substr
	contentStartIdx := width

	for idx, r := range substr[width:] {
		if Symbol(r) != SymbolCode {
			// idx is relative to the substr[width:]
			// so we need to ajust with width
			contentStartIdx = idx + width
			break
		}

		openTagLen++
	}

	// if all symbols after the initial one are SymbolCode, then the loop above will never break and
	// assign correct value to the contentStartIdx, os in this case contentStartIdx will still equal to
	// the width. if this is the case, we set it to the length of the substring
	if contentStartIdx == width {
		contentStartIdx = n
	}

	// index of the last SymbolCode in the closing tag
	lastClosingSymIndex := -1

	// len of longest sequence of SymbolCodes occured
	closingTagLen := 0

	// FIXME: wrong N+1 rule implementation
	for idx, r := range substr[contentStartIdx:] {
		// if the current rune is Symbol, increment the length counter
		if Symbol(r) == SymbolCode {
			closingTagLen++

			// if the len counter is equal to the openTagLen, then we've found the index
			// of the last code symbol in the closing tag
			if closingTagLen == openTagLen {
				lastClosingSymIndex = idx + contentStartIdx
				break
			}
		} else {
			// else reset the counter
			closingTagLen = 0
		}
	}

	// if the code block is considered unclosed, that is, we haven't found SymbolCode sequence
	// with the same length as the starting one before the end of the string, we:
	//   - we return tag as plain text and continue tokenize the rest of the string
	//   - or save the substring as a code block if the tag has length >= 3, that is, opens a code block

	if lastClosingSymIndex == -1 {
		var nodeType NodeType

		isBlockCode := openTagLen >= 3

		// return the opening tag as a plain text if the tag is inline
		if !isBlockCode {
			token = Token{
				Type: TypeText,
				Pos:  i,
				Len:  contentStartIdx,
				Val:  substr[:contentStartIdx],
			}

			nodeType = NodeText

			// signaling the main loop that we have processed only the opening tag
			stride = contentStartIdx
		} else {
			token = Token{
				Type: TypeCodeBlock,
				Pos:  i,
				Len:  n,
				Val:  substr,
			}

			// signaling to the main loop that we've proccessed all the remaining bytes
			stride = n
		}

		warnings = []Warning{{
			Node:        nodeType,
			Index:       i,
			Issue:       IssueUnclosedTag,
			Description: fmt.Sprintf("Unclosed code block at index %d", i),
		}}

		return
	}

	// otherwise we choose Type based on the length of the opening sequence

	var t Type

	if openTagLen >= 3 {
		t = TypeCodeBlock
	} else {
		t = TypeCodeInline
	}

	// assuming all code symbols are the same we won't calculate the width of the last
	// symbol, but will use the 'width' param instead
	codeEnd := lastClosingSymIndex + width

	token = Token{
		Type: t,
		Pos:  i,
		Len:  codeEnd,
		Val:  substr[:codeEnd],
	}

	// adjusting main loop pointer offset
	stride = codeEnd
	return
}
