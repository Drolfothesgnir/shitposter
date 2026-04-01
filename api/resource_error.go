package api

import (
	"errors"
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
)

type ResourceError struct {
	Kind   Kind   `json:"kind"`
	Reason string `json:"reason"`
	Status int    `json:"status"`
	Error  string `json:"error"`
	opErr  *db.OpError
}

func (e ResourceError) StatusCode() int {
	return e.Status
}

func newResourceError(err error) ResourceError {
	var opErr *db.OpError
	if errors.As(err, &opErr) {
		msg := opErr.Err.Error()
		if opErr.Kind == db.KindInternal {
			msg = "an internal error occurred"
		}
		return ResourceError{
			Kind:   KindResource,
			Reason: opErr.Kind.String(),
			Status: opKindToHTTPStatus(opErr.Kind),
			Error:  msg,
			opErr:  opErr,
		}
	}

	return internalResourceError()
}

func notFoundResourceError(msg string) ResourceError {
	return ResourceError{
		Kind:   KindResource,
		Reason: db.KindNotFound.String(),
		Status: http.StatusNotFound,
		Error:  msg,
	}
}

func internalResourceError() ResourceError {
	return ResourceError{
		Kind:   KindResource,
		Reason: db.KindInternal.String(),
		Status: http.StatusInternalServerError,
		Error:  "an internal error occurred",
	}
}

func opKindToHTTPStatus(kind db.Kind) int {
	switch kind {
	case db.KindNotFound:
		return http.StatusNotFound
	case db.KindRelation:
		return http.StatusBadRequest
	case db.KindDeleted:
		return http.StatusGone
	case db.KindConflict:
		return http.StatusConflict
	case db.KindPermission:
		return http.StatusForbidden
	case db.KindConstraint:
		return http.StatusUnprocessableEntity
	case db.KindInvalid:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
