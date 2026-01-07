package scum

import (
	"errors"
	"fmt"
)

// ConfigError describes an error which occures during the configuration of the [Dictionary], like improper Tags.
type ConfigError struct {
	Issue Issue // Issue is a kind or the problem occured.
	Err   error // Err contains original error created during some configuration process.
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}
func (e *ConfigError) Error() string {
	return fmt.Sprintf("%d: %v", e.Issue, e.Err)
}

// NewConfigError is a factory function for creating a *ConfigError.
func NewConfigError(issue Issue, err error) *ConfigError {
	return &ConfigError{
		Issue: issue,
		Err:   err,
	}
}

func newEmptySequenceError() error {
	return NewConfigError(IssueInvalidTagSeqLen, errors.New("provided Tag byte sequence is empty"))
}

func newDuplicateTagIDError(id byte) error {
	return NewConfigError(IssueDuplicateTagID, fmt.Errorf("Tag with ID %d already registered", id))
}

func newUnprintableError(ent string, char byte) error {
	return NewConfigError(
		IssueUnprintableChar,
		fmt.Errorf("%s expected to be 1-byte long ASCII printable character, got %q", ent, char),
	)
}
