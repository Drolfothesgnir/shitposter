package api

import (
	"errors"
	"fmt"
	"net/http"
)

// Flavor represents the type of the core issue
type Flavor string

const (
	FlavorInternal             Flavor = "internal error"
	FlavorBodyTooLarge         Flavor = "body too large"
	FlavorJSONSyntaxError      Flavor = "JSON syntax error"
	FlavorMalformedJSON        Flavor = "malformed JSON"
	FlavorEmptyBody            Flavor = "empty body"
	FlavorInvalidArguments     Flavor = "invalid arguments"
	FlavorIncorrectContentType Flavor = "incorrect content type"
)

// Issue describes the error of a particular payload field
type Issue struct {
	FieldName string `json:"field_name"`
	// Tag is a name of the criterion which the actual payload field doesn't match
	// (e.g. "min" for length of the string or the value of number, or "required" if the field must have non-null value).
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// Vomit describes the error which occured during the parsing or validating of the request body.
type Vomit struct {
	Kind       Kind    `json:"kind"`
	Reason     Flavor  `json:"reason"`
	Status     int     `json:"status"`
	ErrMessage string  `json:"error"`
	Issues     []Issue `json:"issues,omitempty"`
	err        error
}

func (v *Vomit) Error() string {
	return fmt.Sprintf("Payload error: reason: %s; message: %s", v.Reason, v.ErrMessage)
}

func (v *Vomit) Unwrap() error {
	return v.err
}

// puke helps You create a *Vomit.
func puke(reason Flavor, status int, msg string, err error, issues ...Issue) *Vomit {
	return &Vomit{
		err:        err,
		Kind:       KindPayload,
		Reason:     reason,
		Status:     status,
		ErrMessage: msg,
		Issues:     issues,
	}
}

// packTheSpew explicitely transforms the error assumed to be a *Vomit into actual *Vomit.
// Returns the Vomit-internal error in case the err was not of type [Vomit].
func packTheSpew(err error) *Vomit {
	var vErr *Vomit

	if errors.As(err, &vErr) {
		// 1. Dereference the pointer to create a shallow copy
		clone := *vErr

		// 2. Overwrite the internal 'err' with the TOP-LEVEL wrapped error
		// This preserves the full chain for logging/debugging
		clone.err = err

		// 3. Return a pointer to the clone
		return &clone
	}

	// Fallback for non-Vomit errors
	return puke(FlavorInternal, http.StatusInternalServerError, "internal error", err)
}
