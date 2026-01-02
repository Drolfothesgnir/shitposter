package scum

import "fmt"

// AddUniversalTag adds a universal [Tag] to the [Dictionary] and returns
// [ConfigError] if any issues occur during the process.
func (d *Dictionary) AddUniversalTag(name string, seq []byte, greed Greed, rule Rule) error {

	err := checkTagName(name)
	if err != nil {
		return err
	}

	ts, err := NewTagSequence(seq)

	if err != nil {
		return err
	}

	id := ts.ID()

	// check if the Tag is unique
	if d.tags[id].ID() != 0 {
		return NewConfigError(IssueDuplicateTagID, fmt.Errorf("Tag with ID %d already registered", id))
	}

	isSingleChar := ts.Len == 1

	// check rules and greed
	err = checkTagConsistency(isSingleChar, true, rule, greed)
	if err != nil {
		return err
	}

	d.tags[id] = Tag{
		Name:    name,
		Greed:   greed,
		Seq:     ts,
		Rule:    rule,
		OpenID:  id,
		CloseID: id,
	}

	// TODO: add a proper Action based on whther Tag is single-char, greed and rule

	return nil
}
