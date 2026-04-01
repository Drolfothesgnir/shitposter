package api

import "net/http"

const (
	AuthInternal             Flavor = "AUTH_INTERNAL_ERR"
	AuthHeaderNotProvided    Flavor = "AUTH_NO_HEADER"
	AuthInvalidHeaderFormat  Flavor = "AUTH_HEADER_INVALID_FORMAT"
	AuthTypeUnsupported      Flavor = "AUTH_TYPE_UNSUPPORTED"
	AuthAccessTokenErr       Flavor = "AUTH_ACCESS_TOKEN_ERR"
	AuthRefreshTokenErr      Flavor = "AUTH_REFRESH_TOKEN_ERR"
	AuthSessionBlocked       Flavor = "AUTH_SESSION_BLOCKED"
	AuthSessionIncorrectUser Flavor = "AUTH_SESSION_INCORRECT_USER"
	AuthSessionExpired       Flavor = "AUTH_SESSION_EXPIRED"
)

// AuthError describes issues related to access tokens and sessions
type AuthError struct {
	Kind       Kind   `json:"kind"`
	Reason     Flavor `json:"reason"`
	Status     int    `json:"status"`
	ErrMessage string `json:"error"`
}

func (e AuthError) StatusCode() int {
	return http.StatusUnauthorized
}

func newAuthError(reason Flavor, msg string) AuthError {
	return AuthError{
		Kind:       KindAuth,
		Reason:     reason,
		Status:     http.StatusUnauthorized,
		ErrMessage: msg,
	}
}
