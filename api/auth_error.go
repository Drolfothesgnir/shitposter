package api

import "fmt"

const (
	AuthHeaderNotProvided    Flavor = "AUTH_NO_HEADER"
	AuthInvalidHeaderFormat  Flavor = "AUTH_HEADER_INVALID_FORMAT"
	AuthTypeUnsupported      Flavor = "AUTH_TYPE_UNSUPPORTED"
	AuthAccessTokenErr       Flavor = "AUTH_ACCESS_TOKEN_ERR"
	AuthRefreshTokenErr      Flavor = "AUTH_REFRESH_TOKEN_ERR"
	AuthSessionBlocked       Flavor = "AUTH_SESSION_BLOCKED"
	AuthSessionIncorrectUser Flavor = "AUTH_SESSION_INCORRECT_USER"
	AuthSessionExpired       Flavor = "AUTH_SESSION_EXPIRED"
	AuthSessionNotFound      Flavor = "AUTH_SESSION_NOT_FOUND"
	AuthVerificationFailed   Flavor = "AUTH_VERIFICATION_FAILED"
)

// AuthError describes issues related to access tokens and sessions
type AuthError struct {
	Kind       Kind   `json:"kind"`
	Reason     Flavor `json:"reason"`
	Status     int    `json:"status"`
	ErrMessage string `json:"error"`
	err        error  // Keep the original error chain for logs!
}

// Implement the standard Go error interface
func (e *AuthError) Error() string {
	return fmt.Sprintf("Auth error: reason: %s; message: %s", e.Reason, e.ErrMessage)
}

// Allow errors.Is and errors.As to unwrap this error
func (e *AuthError) Unwrap() error {
	return e.err
}

func (e *AuthError) StatusCode() int {
	return e.Status
}

// Accept status and the root error
func newAuthError(reason Flavor, status int, msg string, err error) *AuthError {
	return &AuthError{
		err:        err,
		Kind:       KindAuth, // Hardcoded to guarantee consistency
		Reason:     reason,
		Status:     status,
		ErrMessage: msg,
	}
}
