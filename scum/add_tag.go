package scum

import (
	"errors"
	"fmt"
)

func (d *Dictionary) AddTag(name string, seq []byte, opts ...TagDecorator) error {
	t := Tag{}

	if name == "" {
		// I'll do proper errors later
		return fmt.Errorf("invalid tag name: %q", name)
	}

	n := len(seq)

	if n == 0 {
		return errors.New("tag's byte sequence is empty")
	}

	if n > MaxTagLength {
		return fmt.Errorf("expected the tag's byte sequence to be at most %d bytes long, got %d bytes", MaxTagLength, n)
	}

	if d.Actions[seq[0]] != nil {
		return fmt.Errorf("special symbol with id %d already registered.", seq[0])
	}

	for i := range n {
		if !isASCIIPrintable(seq[i]) {
			return fmt.Errorf("unprintable character in the tag's byte sequence: %q at index %d", seq[i], i)
		}
	}

	t.ID = seq[0]
	t.Name = name
	t.Seq = append([]byte(nil), seq...)

	for _, op := range opts {
		err := op(&t)
		if err != nil {
			return err
		}
	}

	var (
		act Action
		err error
	)

	if t.Len() > 0 {
		// if multi byte

	} else {
		// else the Tag must be single
		act, err = createSingleCharAction(&t)
	}

	if err != nil {
		return err
	}

	d.Actions[seq[0]] = act

	return nil
}
