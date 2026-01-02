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
//     5.2 1 - Intra-word Rule. For single-byte universal non-greedy tags, the rule is evaluated by inspecting the adjacent Unicode runes in the input string.
//     Tokenization state and previous tokens do not affect this rule.
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
//     a Tag an opening tag, you have to set its ClosID value to something other than 0, to inform the Parser that this tag has to be closed with some other Tag.
//     The same with the closing tags: just set the OpenID value to something. To make a Tag universal you have to set both values to the Tag's ID.
//
//     6.1. Example - The Tag expected to be closed with specific other Tag: Imagine you've defined the non-greedy single-byte Tag with name "LINK_TEXT_START",
//     ID '[' and ClosID ']'. Then you've defined the non-greedy Tag with name "LINK_TEXT_END", ID ']' and OpenID '['. During the parsing of the tokens, the Parser,
//     when first encounters the "LINK_TEXT_START", saves its ID to the stack. While the '[' is at the top of the stack and the Parser encounters a closing Tag
//     it checks if the encountered Tag ID is equal to the "LINK_TEXT_START's" ClosID. What will happen next is a good question.
//
// Attributes.
//
//   - Each Tag can have valued or flag attributes. You can define a Tag's attribute by creating other special Tags. To do this you need to define a
//     single-byte symbol which marks the start of the Attribute. You also need to define the opening and closing tags for the Attribute's body.
//     The valued Attribute definition - <marker>(name)<body start>(content)<body end>.
//
//     The flag Attribute definition - <marker><body start>(name)<body end>.
//
//     Example: tou have a Tag with name "TAG", ID '[' and a closing Tag for it with ID ']'. You define the Attribute marker as '!' and '{' and '}' as
//     the body opening and closing tags respectively. You use it by writing "[hello world]!attr1{foo-bar-001}!attr2{goodbye world}". You cam also create flag
//     Attributes like this: [...]!{flagAttr1}!{flagAttr2}. You can combine valued and flag attributes. All attributes, that are following immediately after a Tag,
//     will be considered this Tag's attributes.
//
// Escape symbol.
//
//   - You can define an Escape symbol, which, when encountered during the tokenization, will make the Tokenizer treat next character, whether it's special or
//     a simple text character, as a plain text.
//
// TODO: add preallocated tag strings inside the dictionary
// TODO: add docs for cases when closing tag does not match the opening and is returned as plain text along with a Warning
package scum

// Dictionary manages creation and deletion of Tags and their corresponding Actions.
type Dictionary struct {
	// actions maps particular Tag's ID to its corresponding [Action].
	actions [256]Action

	// tags maps particular Tag's ID to its Tag's info.
	tags [256]Tag
}

// Tag allows to get particular Tag's info by providing its ID.
func (d *Dictionary) Tag(id byte) (Tag, bool) {
	t := d.tags[id]
	return t, t.Seq.Len != 0
}

// Action allows to get particular Tag's [Action] by providing the Tag's ID.
func (d *Dictionary) Action(id byte) (Action, bool) {
	a := d.actions[id]
	return a, a != nil
}

// func checkMultiByteTagConsistency(name string, seq []byte, tokenType TokenType, input string, i int, warns *[]Warning) (token Token, stride int, ok bool) {
// 	n := len(input)
// 	// 1. Checking the tag consistency

// 	// 1.1 Checking if the input after index i is similar to the rest of the tag byte sequence
// 	// we start comparing from i+1 because byte at index i is the same, since it was an Action trigger
// 	diffIndex, substrShorter := checkByteDifference(input[i+1:], seq[1:])

// 	// if the input's tag sequence is different from seq, that is diffIndex > -1, return [Token] with
// 	// type [TokenText] and add a Warning
// 	if diffIndex > -1 {
// 		// adjusting the relative diffIndex
// 		absDiffIndex := diffIndex + i + 1

// 		matchedSeq := input[i:absDiffIndex]

// 		token = Token{
// 			Type:  TokenText,
// 			TagID: seq[0],
// 			Pos:   i,
// 			Width: len(matchedSeq),
// 			Raw:   matchedSeq,
// 			Inner: matchedSeq,
// 		}

// 		wrong, _, valid := extractNextRune(input[absDiffIndex:])

// 		quoted := strconv.QuoteRune(wrong)

// 		got := ""

// 		if !valid {
// 			got = "unrecognizable character "
// 		}

// 		tagDesc := " "

// 		desc := "Unexpected symbol at index " +
// 			strconv.Itoa(absDiffIndex) +
// 			" while interpreting the" + tagDesc + "tag with name '" +
// 			name + "': expected to get '" + string(seq[diffIndex+1]) +
// 			"', got " + got + quoted + "."

// 		*warns = append(*warns, Warning{
// 			Issue:       IssueUnexpectedSymbol,
// 			Pos:         absDiffIndex,
// 			Description: desc,
// 		})

// 		// we've processed all bytes from the index i and up to index of the first divergence
// 		stride = absDiffIndex - i

// 		ok = true
// 		return
// 	}

// 	// 1.2 Checking if the input string ended before completing the opening tag sequence, that is substrShorter == true

// 	// in this case we return the unfinished tag as [Token] with type [TokenText], and a [Warning]
// 	if substrShorter {
// 		matchedSeq := input[i:n]

// 		matchedSeqLen := len(matchedSeq)

// 		token = Token{
// 			Type:  TokenText,
// 			TagID: seq[0],
// 			Pos:   i,
// 			Width: matchedSeqLen,
// 			Raw:   matchedSeq,
// 			Inner: matchedSeq,
// 		}

// 		// seq[n-i:] is a sub sequence of seq starting from the first missing byte from the string
// 		desc := "Unexpected end of the line while interpreting the opening tag with name '" +
// 			name + "': expected to get '" + string(seq[n-i:]) + "' but got EOL."

// 		*warns = append(*warns, Warning{
// 			Issue:       IssueUnexpectedEOL,
// 			Pos:         n,
// 			Description: desc,
// 		})

// 		// we've processed all bytes from i to the end of the input
// 		stride = matchedSeqLen

// 		ok = true
// 		return
// 	}

// 	return
// }
