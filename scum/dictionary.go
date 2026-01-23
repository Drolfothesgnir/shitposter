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
//  2. Tags can contain at most [MaxTagLen] symbols. Escape and Attribute signature tags can contain only 1 symbol.
//  3. ALL special symbols must be 1-byte long printable ASCII characters.
//  4. Nested tags with the same ID will have no effect. Children of the repeated descendants will become
//     children of the "oldest" original tag and the duplicates will not end up in the final AST.
//  5. A universal Tag is one, which has both the opening and the closing tags the same.
//  6. The parser will try to make sense out of the User's gibberish and will not return any errors but only a slice of [Warning].
//  7. The trigger Tag is one which starts the Action.
//  8. The complementary Tag is one which closes the greedy Tag's body.
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
//     WARNING: Closing tag is any run of the trigger symbol whose length equals the opening run length k.
//     To avoid accidental closure, choose k such that no run of length k appears inside the content.
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
//     single-byte symbol which marks the start of the Attribute. You also need to define the opening and closing tags for the Attribute's payload.
//     The valued Attribute definition - <marker>(name)<payload start>(content)<payload end>.
//
//     The flag Attribute definition - <marker><payload start>(name)<payload end>.
//
//     Example: tou have a Tag with name "TAG", ID '[' and a closing Tag for it with ID ']'. You define the Attribute marker as '!' and '{' and '}' as
//     the payload opening and closing tags respectively. You use it by writing "[hello world]!attr1{foo-bar-001}!attr2{goodbye world}". You cam also create flag
//     Attributes like this: [...]!{flagAttr1}!{flagAttr2}. You can combine valued and flag attributes. All attributes, that are following immediately after a Tag,
//     will be considered this Tag's attributes.
//
//     Attribute will be attached to previous Tag, whether it's a normal tag or a text.
//
//     Example: in the input "$$hello$$ world!STYLE{color: \"#fff\"}", the STYLE attribute will be attached to the text node " world".
//
//     You also can have escaping inside the attribute's payload. For this you need to define an escape symbol via [Dictionary.SetEscapeTrigger].
//     Also escaping the Attribute trigger will result in not starting the Attribute processing.
//
// Escape symbol.
//
//   - You can define an Escape symbol, which, when encountered during the tokenization, will make the Tokenizer treat next character, whether it's special or
//     a simple text character, as a plain text. Escape symbol can be only 1-byte long ASCII char.
//     In case of the escape symbol being before non-special character, or being the last symbol in the input, it will be treated as a plain text,
//     and a Warning will be returned. Escaping also available inside the attribute payload bodies. As for now, escape inside the payload body
//     will not cause any Warnings even if it's placed before a non-special character. Live with it. As for now, escaping is not available inside greedy Tag's
//     body. Live with it, too.
//
// Scanning limits.
//
//   - Greedy Tags have limited length of the payload, defined by [Limits.MaxPayloadLen]. If after the reaching the maximum payload length,
//     the complementary Tag was not found, the trigger Tag will be skipped as a plain text and a Warning of unclosed Greedy Tag will be added.
//     If the provided limit is 0, then the actual limit value will be [DefaultMaxPayloadLen].
//
//   - Tag-Vs-Content-rule-based Tags have length limits for opening and closing sequences and for the payload. [Limits.MaxKeyLen] defines
//     the sequences limit and [Limits.MaxPayloadLen] defines the payload limit. If the opening Tag sequence is larger than the provided
//     limit, the opening sequence will be trated as a plain text and a Warning will be added. The payload limit logic is the same as for
//     the greedy Tags. If either the [Limits.MaxKeyLen] or the [Limits.MaxPayloadLen] are provided as 0, they will be replaced with
//     [DefaultMaxKeyLen] and [DefaultMaxPayloadLen] respectively.
//
//   - Attributes have limits for the key and the payload. [Limits.MaxAttrKeyLen] defines the limit for the key and [Limits.MaxAttrPayloadLen]
//     defines the limit for the payload. If the attribute key length exceeds the limit without finding the payload start symbol, the attribute
//     trigger is treated as plain text and a Warning with [IssueAttrKeyTooLong] is added. If the attribute payload length exceeds the limit
//     without finding the payload end symbol, the attribute trigger is treated as plain text and a Warning with [IssueAttrPayloadTooLong] is added.
//
// # Config errors.
//
// A [ConfigError] is returned during configuration when invalid parameters are provided. Each ConfigError contains an [Issue] describing
// the kind of problem encountered.
//
//   - [NewDictionary]: Returns [IssueNegativeLimit] if any field in [Limits] is negative.
//
//   - [NewWarnings]: Returns [IssueNegativeWarningsCap] if the provided capacity is negative.
//
//   - [Dictionary.SetEscapeTrigger]: Returns [IssueUnprintableChar] if the escape symbol is not a printable ASCII character.
//     Returns [IssueDuplicateTagID] if the symbol is already registered as a Tag or other special symbol.
//
//   - [Dictionary.SetAttributeSignature]: Returns [IssueDuplicateTagID] if the trigger symbol is already registered.
//     Returns [IssueInvalidAttrSymbol] if the trigger symbol equals the payload start or payload end symbol.
//     Returns [IssueUnprintableChar] if any of the three symbols (trigger, payload start, payload end) is not a printable ASCII character.
//
//   - [NewTagSequence]: Returns [IssueInvalidTagSeqLen] if the byte sequence is empty or longer than [MaxTagLen].
//     Returns [IssueUnprintableChar] if any byte in the sequence is not a printable ASCII character.
//
//   - Tag name validation: Returns [IssueInvalidTagNameLen] if the tag name is empty or longer than [MaxTagNameLen] UTF-8 characters.
//
//   - Tag consistency validation: Returns [IssueInvalidRule] if the rule value exceeds [MaxRule], or if the rule is incompatible with
//     the tag's greed level (e.g., [RuleInfraWord] requires [NonGreedy], [RuleTagVsContent] requires greedy tag).
//     Returns [IssueInvalidGreedLevel] if the greed level exceeds [MaxGreedLevel].
//     Returns [IssueRuleInapplicable] if a rule other than [RuleNA] is applied to a non-single-char or non-universal tag.
//
//   - Tag registration ([Dictionary.AddTag], [Dictionary.AddUniversalTag]): Returns [IssueDuplicateTagID] if the tag ID is already registered.
//
// # Warnings.
//
// A [Warning] is added during tokenization when the input contains problematic but recoverable patterns. Warnings do not stop processing;
// instead, the tokenizer attempts to make sense of the input. Each Warning contains an [Issue] and a position in the input.
//
//   - [IssueUnexpectedEOL]: Added when a special symbol is found at the very end of the input where more content is expected.
//     This includes: escape symbol at EOL, opening tag at EOL, attribute trigger at EOL, and attribute payload start at EOL.
//
//   - [IssueRedundantEscape]: Added when the escape symbol precedes a non-special character. The escaped character is still
//     included in the output as an escape sequence token.
//
//   - [IssueUnclosedTag]: Added when a greedy or grasping tag's opening sequence is found, but no matching closing sequence
//     exists in the input. For [Greedy] tags, the opening tag is skipped as plain text. For [Grasping] tags, the entire rest
//     of the input becomes the tag's payload.
//
//   - [IssueMisplacedClosingTag]: Added when a closing tag is found at the very beginning of the input (index 0). The closing
//     tag is treated as plain text.
//
//   - [IssueUnexpectedSymbol]: Added when a multi-char tag's sequence is interrupted by an unexpected byte. The partial sequence
//     is treated as plain text.
//
//   - [IssueTagKeyTooLong]: Added when a [RuleTagVsContent] tag's opening sequence exceeds [Limits.MaxKeyLen] bytes.
//     The opening sequence is treated as plain text.
//
//   - [IssueTagPayloadTooLong]: Added when a [Greedy] tag's payload exceeds [Limits.MaxPayloadLen] bytes without
//     finding the closing sequence. The tag is treated according to its [Greed] level.
//
//   - [IssueAttrKeyTooLong]: Added when the attribute payload start symbol is not found within [Limits.MaxAttrKeyLen] bytes
//     after the attribute trigger. The trigger is treated as plain text.
//
//   - [IssueAttrPayloadTooLong]: Added when the attribute payload end symbol is not found within [Limits.MaxAttrPayloadLen] bytes
//     after the payload start. The trigger is treated as plain text.
//
//   - [IssueUnclosedAttrPayload]: Added when the attribute payload start symbol is found but the payload end symbol is missing
//     before the end of the input. The trigger is treated as plain text.
//
//   - [IssueEmptyAttrPayload]: Added when the attribute payload is present but empty (e.g., "!k{}" or "!{}"). The trigger is
//     treated as plain text.
//
//   - [IssueWarningsTruncated]: Added automatically by [Warnings] when using [WarnOverflowTrunc] policy and the maximum capacity
//     is reached. This warning replaces all subsequent warnings and indicates how many were dropped.
//
// TODO: add preallocated tag strings inside the dictionary
// TODO: add docs for cases when closing tag does not match the opening and is returned as plain text along with a Warning
package scum

// Dictionary manages creation and deletion of Tags and their corresponding Actions.
type Dictionary struct {
	// Limits is a set of numbers defined to limit the excessive input scanning
	// during the tokenization process and to reduce the damage of the potential DoS attacks.
	Limits Limits

	// actions maps particular Tag's ID to its corresponding [Action].
	actions [256]Action

	// tags maps particular Tag's ID to its Tag's info.
	tags [256]Tag

	// attrTrigger is a special symbol, which starts Attribute Action.
	attrTrigger byte

	// attrPayloadStart is special symbol, which marks the start of the Attribute's payload.
	attrPayloadStart byte

	// attrPayloadEnd is a special symbol, which masrks the end of the Attribute's payload.
	attrPayloadEnd byte

	// escapeTrigger is a symbol which make the tokenizer treat the next symbol after it as non-special.
	escapeTrigger byte
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

// IsSpecial returns true if the provided char is registered inside the [Dictionary]
// as either a [Tag], an attribute signature's part, an escape symbol, or an escape symbol;
func (d *Dictionary) IsSpecial(char byte) bool {
	if char == 0 {
		return false
	}

	switch char {
	case d.attrTrigger,
		d.attrPayloadStart,
		d.attrPayloadEnd,
		d.escapeTrigger:
		return true
	}

	return d.actions[char] != nil
}

// NewDictionary creates new [Dictionary] with provided optional limits.
// [Limits] struct must be provided, but You can fill relevant fields only.
// All zero fields will be populated with default values.
// Negative limits will cause [ConfigError].
func NewDictionary(limits Limits) (Dictionary, error) {
	if err := limits.Validate(); err != nil {
		return Dictionary{}, err
	}

	values := [4]*int{
		&limits.MaxAttrKeyLen,
		&limits.MaxAttrPayloadLen,
		&limits.MaxPayloadLen,
		&limits.MaxKeyLen,
	}

	defaultValues := [4]int{
		DefaultMaxAttrKeyLen,
		DefaultMaxAttrPayloadLen,
		DefaultMaxPayloadLen,
		DefaultMaxKeyLen,
	}

	for i, v := range defaultValues {
		if *values[i] == 0 {
			*values[i] = v
		}
	}

	return Dictionary{Limits: limits}, nil
}
