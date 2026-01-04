package scum

import "errors"

// ConfigError describes an error which occures during the configuration of the [Dictionary], like improper Tags.
type ConfigError struct {
	Issue Issue // Issue is a kind or the problem occured.
	Err   error // Err contains original error created during some configuration process.
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}
func (e *ConfigError) Error() string {
	return e.Err.Error()
}

// NewConfigError is a factory function for creating a *ConfigError.
func NewConfigError(issue Issue, err error) *ConfigError {
	return &ConfigError{
		Issue: issue,
		Err:   err,
	}
}

func newEmptySequenceError() error {
	return NewConfigError(IssueInvalidTagSeq, errors.New("provided Tag byte sequence is empty"))
}
