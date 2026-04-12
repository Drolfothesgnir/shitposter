package sml

import "fmt"

const (
	ReasonInternal      = "INTERNAL"
	ReasonInvalidParams = "INVALID_PARAMS"
)

type ConfigError struct {
	SubjectName string `json:"subject_name"`
	Reason      string `json:"reason"`
	Err         error  `json:"error"`
	// some other stuff...
}

func (e ConfigError) Error() string {
	return fmt.Sprintf("error during creation of the %s; reason - %s; message - %s", e.SubjectName, e.Reason, e.Err.Error())
}

func (e ConfigError) Unwrap() error {
	return e.Err
}

func NewConfigError(name, reason string, err error) *ConfigError {
	return &ConfigError{
		SubjectName: name,
		Reason:      reason,
		Err:         err,
	}
}
