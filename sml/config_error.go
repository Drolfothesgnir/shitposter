package sml

import "fmt"

const (
	ReasonInternal      = "INTERNAL"
	ReasonInvalidParams = "INVALID_PARAMS"
)

// ConfigError occures when the SML parser - [Eater], or other structures, which allow configuration,
// have some issues because of the config.
type ConfigError struct {
	// SubjectName is the name of the structure which which creation was attempted, e.g. "SML Parser".
	SubjectName string `json:"subject_name"`
	Reason      string `json:"reason"`
	Err         error  `json:"error"`
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
