package api

import "net/http"

type AuthError struct {
	Kind  Kind   `json:"kind"`
	Error string `json:"error"`
}

func (e AuthError) StatusCode() int {
	return http.StatusUnauthorized
}

func newAuthError(msg string) AuthError {
	return AuthError{
		Kind:  KindAuth,
		Error: msg,
	}
}
