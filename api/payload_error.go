package api

import (
	"fmt"
	"net/http"
	"strings"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
)

type PayloadError struct {
	Kind   Kind    `json:"kind"`
	Error  string  `json:"error"`
	Issues []Issue `json:"issues,omitempty"`
}

func (pe PayloadError) StatusCode() int {
	return http.StatusBadRequest
}

func newPayloadError(message string, err error) PayloadError {
	return PayloadError{
		Kind:   KindPayload,
		Error:  message,
		Issues: validationErrorsToIssues(err),
		// TODO: Add Descriptor?
	}
}

func validationErrorsToIssues(err error) []Issue {
	return []Issue{}
}

func getBindingErrorMessage(tag string, value any, param string) string {
	switch tag {
	case "required":
		return "this field is required"

	case "min":
		switch v := value.(type) {
		case int, int8, int16, int32, int64:
			return fmt.Sprintf("value %d is too small, minimum is %s", v, param)

		case string:
			return fmt.Sprintf("value %q is too short (min %s characters)", v, param)

		default:
			return fmt.Sprintf("value is below the allowed minimum: %v", value)
		}

	case "max":
		switch v := value.(type) {
		case int, int8, int16, int32, int64:
			return fmt.Sprintf("value %d is too big, maximum is %s", v, param)

		case string:
			return fmt.Sprintf("value %q is too long (max %s characters)", v, param)

		default:
			return fmt.Sprintf("value exceeds the allowed maximum: %v", value)
		}

	case "len":
		return "invalid length"

	case "email":
		return "invalid email address"

	case "url":
		return "invalid URL format"

	case "alphanum":
		return "must contain only letters and numbers"

	case "alpha":
		return "must contain only letters"

	case "numeric":
		return "must contain only numbers"

	case "gte":
		return "must be greater than or equal to the allowed minimum"

	case "lte":
		return "must be less than or equal to the allowed maximum"

	case "gt":
		return "must be greater than the allowed minimum"

	case "lt":
		return "must be less than the allowed maximum"

	case "oneof":
		return "must be one of the allowed values"

	case "uuid":
		return "invalid UUID format"

	case "ip":
		return "invalid IP address"

	case "ipv4":
		return "invalid IPv4 address"

	case "ipv6":
		return "invalid IPv6 address"

	case "startswith":
		return "must start with the required prefix"

	case "endswith":
		return "must end with the required suffix"

	case "comment_order":
		return fmt.Sprintf(
			"comment order must be one of [%s]",
			strings.Join(db.CommentOrderMethods, ", "),
		)

	default:
		return "invalid input"
	}
}
