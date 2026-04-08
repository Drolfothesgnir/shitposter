package scum

// AddTag registers new [Tag] in the [Dictionary] and return [ConfigError] if the provided values are
// inconsistent or invalid.
func (d *Dictionary) AddTag(name string, seq []byte, greed Greed, rule Rule, openID, closeID byte) error {

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
		return newDuplicateTagIDError(id)
	}

	isSingleChar := ts.Len == 1
	isUniversal := openID == seq[0] && closeID == seq[0]

	// check rules and greed
	err = checkTagConsistency(isSingleChar, isUniversal, rule, greed)
	if err != nil {
		return err
	}

	t := Tag{
		Name:    name,
		Greed:   greed,
		Seq:     ts,
		Rule:    rule,
		OpenID:  openID,
		CloseID: closeID,
	}

	d.tags[id] = t

	d.actions[id] = CreateAction(&t)

	return nil
}
