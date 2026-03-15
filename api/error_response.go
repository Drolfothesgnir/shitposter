package api

import (
	"bytes"
	"encoding/json"
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

// TODO: replace this helper with json.NewDecoder(rec.Body).Decode(&ErrorResponse) in all tests
func extractErrorFromBuffer(buf *bytes.Buffer) (*ErrorResponse, error) {
	var resp ErrorResponse
	if err := json.NewDecoder(buf).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
