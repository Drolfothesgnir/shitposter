package api

import (
	"fmt"
)

// Flavor represents the type of the core issue
type Flavor string

const (
	FlavorInternal          Flavor = "INTERNAL_ERR"
	ReqBodyTooLarge         Flavor = "REQ_BODY_TOO_LARGE"
	ReqJSONSyntaxError      Flavor = "REQ_JSON_SYNTAX_ERR"
	ReqMalformedJSON        Flavor = "REQ_JSON_MALFORMED"
	ReqEmptyBody            Flavor = "REQ_BODY_EMPTY"
	ReqInvalidArguments     Flavor = "REQ_INVALID_ARGUMENTS"
	ReqIncorrectContentType Flavor = "REQ_INVALID_CONTENT_TYPE"
	ReqMissingData          Flavor = "REQ_MISSING_DATA"
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
// TODO: normalize all error schemas in API responses, that is ResourceError, AuthError and Vomit should have same structure
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

// // packTheSpew explicitely transforms the error assumed to be a *Vomit into actual *Vomit.
// // Returns the Vomit-internal error in case the err was not of type [Vomit].
// // TODO: should i even have this function? why can't i just return *Vomit directly from [ingestJSONBody]?
// func packTheSpew(err error) *Vomit {
// 	var vErr *Vomit

// 	if errors.As(err, &vErr) {
// 		// 1. Dereference the pointer to create a shallow copy
// 		clone := *vErr

// 		// 2. Overwrite the internal 'err' with the TOP-LEVEL wrapped error
// 		// This preserves the full chain for logging/debugging
// 		clone.err = err

// 		// 3. Return a pointer to the clone
// 		return &clone
// 	}

// 	// Fallback for non-Vomit errors
// 	return puke(FlavorInternal, http.StatusInternalServerError, "internal error", err)
// }
