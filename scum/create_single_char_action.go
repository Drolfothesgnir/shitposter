package scum

import "fmt"

// createSingleCharAction creates an [Action] for the provided [Tag], based on it's type, opening, closing or universal.
// It will return a [ConfigError] if the provided Tag has inconsitent fields.
func createSingleCharAction(t *Tag) (Action, error) {
	if t.IsUniversal() {
		return createSingleCharUnivAction(t)
	}

	if t.IsOpening() {

	}

	if t.IsClosing() {

	}

	// guard case when the Tag has inconsistent [Tag.OpenID] and [Tag.CloseID] values, that is
	// when the both fields are set but to a different values, which makes Tag's type ambiguous,
	return nil, NewConfigError(
		IssueAmbiguousTagType,
		fmt.Errorf(
			"ambiguous tag's type due to both opening and closing tag IDs being set improperly: opening tag's ID: %q; closing tag's ID: %q",
			t.OpenID, t.CloseID),
	)
}

func singleByteTag(input string, id byte, i int) (token Token, stride int, skip bool) {
	token = Token{
		Type:  TokenTag,
		TagID: id,
		Pos:   i,
		Width: 1,
		Raw:   input[i : i+1],
	}

	stride = 1
	return
}
