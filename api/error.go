package api

import (
	"net/http"
)

type Kind string

const (
	KindAuth     Kind = "auth"
	KindPayload  Kind = "payload"
	KindResource Kind = "resource"
)

type APIError interface {
	StatusCode() int
}

func abortWithError(w http.ResponseWriter, err APIError) {
	respondWithJSON(w, err.StatusCode(), err)
}
