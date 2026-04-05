package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type ingestJSONBodyPayload struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type errReadCloser struct {
	err error
}

func (e errReadCloser) Read(p []byte) (int, error) {
	return 0, e.err
}

func (e errReadCloser) Close() error {
	return nil
}

func TestIngestJSONBody(t *testing.T) {
	readErr := errors.New("boom")

	testCases := []struct {
		name            string
		contentType     string
		body            string
		overrideBody    io.ReadCloser
		wantReason      Flavor
		wantStatus      int
		wantErrMessage  string
		wantTarget      ingestJSONBodyPayload
	}{
		{
			name:           "OK",
			contentType:    "application/json",
			body:           `{"name":"alice","age":30}`,
			wantTarget:     ingestJSONBodyPayload{Name: "alice", Age: 30},
		},
		{
			name:           "OKWithCharset",
			contentType:    "application/json; charset=utf-8",
			body:           `{"name":"alice","age":30}`,
			wantTarget:     ingestJSONBodyPayload{Name: "alice", Age: 30},
		},
		{
			name:           "IncorrectContentType",
			contentType:    "text/plain",
			body:           `{"name":"alice","age":30}`,
			wantReason:     ReqIncorrectContentType,
			wantStatus:     http.StatusUnsupportedMediaType,
			wantErrMessage: "application/json Content-Type expected but got text/plain instead",
		},
		{
			name:           "BodyTooLarge",
			contentType:    "application/json",
			body:           `{"name":"` + strings.Repeat("a", maxBodySize) + `"}`,
			wantReason:     ReqBodyTooLarge,
			wantStatus:     http.StatusRequestEntityTooLarge,
			wantErrMessage: "body too large",
		},
		{
			name:           "JSONSyntaxError",
			contentType:    "application/json",
			body:           `{"name":truue}`,
			wantReason:     ReqJSONSyntaxError,
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "syntax error in JSON",
		},
		{
			name:           "TypeMismatch",
			contentType:    "application/json",
			body:           `{"name":"alice","age":"30"}`,
			wantReason:     ReqMalformedJSON,
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "incorrect data type for field age",
		},
		{
			name:           "UnknownField",
			contentType:    "application/json",
			body:           `{"name":"alice","age":30,"extra":"x"}`,
			wantReason:     ReqMalformedJSON,
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: `json: unknown field "extra"`,
		},
		{
			name:           "EmptyBody",
			contentType:    "application/json",
			body:           ``,
			wantReason:     ReqEmptyBody,
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "the request body is empty",
		},
		{
			name:           "IncompletePayload",
			contentType:    "application/json",
			body:           `{"name":"alice"`,
			wantReason:     ReqMalformedJSON,
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "the request payload stream is interrupted or the body is incomplete",
		},
		{
			name:           "MultipleJSONObjects",
			contentType:    "application/json",
			body:           `{"name":"alice","age":30}{"name":"bob","age":25}`,
			wantReason:     ReqMalformedJSON,
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "request body must only contain a single JSON object",
		},
		{
			name:           "InternalReadError",
			contentType:    "application/json",
			overrideBody:   errReadCloser{err: readErr},
			wantReason:     FlavorInternal,
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: "internal error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(tc.body))
			if tc.overrideBody != nil {
				request.Body = tc.overrideBody
			}
			if tc.contentType != "" {
				request.Header.Set("Content-Type", tc.contentType)
			}

			var target ingestJSONBodyPayload
			vErr := ingestJSONBody(recorder, request, &target)

			if tc.wantReason == "" {
				require.Nil(t, vErr)
				require.Equal(t, tc.wantTarget, target)
				return
			}

			require.NotNil(t, vErr)
			require.Equal(t, KindPayload, vErr.Kind)
			require.Equal(t, tc.wantReason, vErr.Reason)
			require.Equal(t, tc.wantStatus, vErr.Status)
			require.Equal(t, tc.wantErrMessage, vErr.ErrMessage)
		})
	}
}
