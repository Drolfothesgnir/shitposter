package api

import (
	"bytes"
	"encoding/json"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewErrorResponse(err error) ErrorResponse {
	return ErrorResponse{err.Error()}
}

func extractErrorFromBuffer(buf *bytes.Buffer) (*ErrorResponse, error) {
	var resp ErrorResponse
	if err := json.NewDecoder(buf).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
