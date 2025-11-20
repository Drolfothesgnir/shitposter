package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type ErrorField struct {
	FieldName    string `json:"field_name"`
	ErrorMessage string `json:"error_message"`
}

type ErrorResponse struct {
	Error  string       `json:"error"`
	Fields []ErrorField `json:"fields,omitempty"`
}

func NewErrorResponse(err error, fields ...ErrorField) ErrorResponse {
	return ErrorResponse{Error: err.Error(), Fields: fields}
}

func ExtractErrorFields(err error) []ErrorField {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return []ErrorField{}
	}

	fields := make([]ErrorField, len(ve))
	for i, fe := range ve {
		fields[i] = ErrorField{
			FieldName:    fe.Field(),
			ErrorMessage: getBindingErrorMessage(fe.Tag()),
		}
	}

	return fields
}

func getBindingErrorMessage(tag string) string {
	switch tag {
	case "required":
		return "this field is required"

	case "min":
		return "value is too short"

	case "max":
		return "value is too long"

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
			strings.Join(commentOrderMethods, ", "),
		)

	default:
		return "invalid input"
	}
}

func extractErrorFromBuffer(buf *bytes.Buffer) (*ErrorResponse, error) {
	var resp ErrorResponse
	if err := json.NewDecoder(buf).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
