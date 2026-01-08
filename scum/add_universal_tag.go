package scum

// AddUniversalTag adds a universal [Tag] to the [Dictionary] and returns
// [ConfigError] if any issues occur during the process.
func (d *Dictionary) AddUniversalTag(name string, seq []byte, greed Greed, rule Rule) error {
	return d.AddTag(name, seq, greed, rule, seq[0], seq[0])
}
