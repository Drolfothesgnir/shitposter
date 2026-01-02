package scum

// Tag contains all the info about a particular tag, relevant for the tokenizing and parsing.
type Tag struct {
	// Name is the name of the Tag. Does not need to be unique.
	Name string

	// Greed is a measure of how many next characters must the Tag consume until the closing Tag occurs, during the tokenization.
	// Possible levels: 0, 1 and 2. The Tag with the level > 0 is considered a greedy.
	//
	// 0 means the Tag only consumes itself and the result [Token] has only the opening Tag as internal value.
	//
	// 1 means that all characters between the opening and closing Tags will be consumed and become the result
	// Token's internal value. But if the Tag is unclosed, only opening Tag is counsumed, just like with Greed 0,
	// and a [Warning] is added.
	//
	// 2 has the same behaviour as properly closed Greed-1 Tag, even when the closing Tag is missing. The Warning will still be added.
	Greed Greed

	// Seq defines the sequence of bytes from which the Tag's string consists.
	Seq TagSequence

	// Rule defines optional behaviour for the universal single-byte Tags during the tokenization process.
	//
	// Possible values are 0, 1 and 2.
	//
	// 0 means no additional behavior.
	//
	// 1 - the Intra-Word rule. Can be used by only a non-greedy Tag. If the rule is applied, the Tag's trigger symbol will
	// be considered a plain-text if it has an alphanumerical character, a punctuation character or the
	// same Tag trigger symbol on each side.
	//
	// 2 - the Tag-VS-Content rule. Can be used by only a greedy Tag. The rule is used to avoid confusion of the plain text and the closing tag
	// during the tokenization process. The rule allows to a universal single-byte Tag to have variable width of the opening and closing sequences.
	// There are 2 conditions for this rule to work:
	//  1. The width of the opening and closing Tags must be the same.
	//  2. The widths of the opening and closing Tags must differ from the width of the sequence of the Tag symbol in the content.
	//
	//  - Example:
	//		You've defined the Tag with ID '&', name "CODE" and Rule with value 2. You have string "&&&const T = a && b;&&&". In this case
	//    the "CODE" will capture "const T = a && b;" as it's content, even though the Tag is single and there is "&&" in the content.
	Rule Rule

	// OpenID is the ID of the Tag which is meant to be an opening Tag for this one.
	//
	// Set both OpenID and CloseID to the ID of this Tag to make it universal.
	// WARNING: You have to set at least on of the OpenID or CloseID for the Parser to consider the Tag valid.
	OpenID byte

	// CloseID is the ID of the Tag which is meant to be a closing Tag for this one.
	//
	// Set both OpenID and CloseID to the ID of this Tag to make it universal.
	// WARNING: You have to set at least on of the OpenID or CloseID for the Parser to consider the Tag valid.
	CloseID byte
}

// ID is unique byte value used as opening byte for the Tag's sequence
func (t *Tag) ID() byte {
	return t.Seq.ID()
}

// IsUniversal returns true when the Tag's signature is the same as it's closing [Tag].
func (t *Tag) IsUniversal() bool {
	return t.CloseID == t.ID() && t.OpenID == t.ID()
}

func (t *Tag) IsOpening() bool {
	return t.CloseID != 0 && t.OpenID == 0
}

func (t *Tag) IsClosing() bool {
	return t.OpenID != 0 && t.CloseID == 0
}

func (t *Tag) Len() uint8 {
	return t.Seq.Len
}

// TagDecorator is a decorator function which allows to fill optional fields of the [Tag].
type TagDecorator func(t *Tag)

func WithGreed(greed Greed) TagDecorator {
	return func(t *Tag) {
		t.Greed = greed
	}
}

func WithRule(rule Rule) TagDecorator {
	return func(t *Tag) {
		t.Rule = rule
	}
}

// NewTag creates new [Tag] from the sequence of bytes, name opening/closing Tag IDs and
// optional values.
func NewTag(seq []byte, name string, openID, closeID byte, opts ...TagDecorator) (Tag, error) {
	s, err := NewTagSequence(seq)

	if err != nil {
		return Tag{}, err
	}

	t := Tag{
		Name:    name,
		Seq:     s,
		OpenID:  openID,
		CloseID: closeID,
	}

	for _, dec := range opts {
		dec(&t)
	}

	return t, nil
}
