package scum

// AddUniversalTag adds a universal [Tag] to the [Dictionary] and returns
// [ConfigError] if any issues occur during the process.
func (d *Dictionary) AddUniversalTag(name string, seq []byte, greed Greed, rule Rule) error {

	err := checkTagName(name)
	if err != nil {
		return err
	}

	err = checkTagBytes(d, seq)
	if err != nil {
		return err
	}

	// TODO: check greed and rules

	d.tags[seq[0]] = Tag{
		ID:      seq[0],
		Name:    name,
		Greed:   greed,
		Seq:     seq,
		Rule:    rule,
		OpenID:  seq[0],
		CloseID: seq[0],
	}

	// TODO: add a proper Action

	return nil
}
