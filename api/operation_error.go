package api

import (
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
)

type OperationError struct {
	Kind   Kind   `json:"kind"`
	Reason string `json:"reason"`
	Error  string `json:"error"`
	status int
}

func (e OperationError) StatusCode() int {
	return e.status
}

func newOperationError(opErr *db.OpError) OperationError {
	msg := opErr.Err.Error()
	if opErr.Kind == db.KindInternal {
		msg = "an internal error occurred"
	}

	return OperationError{
		Kind:   KindOperation,
		Reason: opErr.Kind.String(),
		Error:  msg,
		status: opKindToHTTPStatus(opErr.Kind),
	}
}

func internalOperationError() OperationError {
	return OperationError{
		Kind:   KindOperation,
		Reason: db.KindInternal.String(),
		Error:  "an internal error occurred",
		status: http.StatusInternalServerError,
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
