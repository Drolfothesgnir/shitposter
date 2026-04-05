package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxBodySize = 1 << 20
const contentJSON = "application/json"

func ingestJSONBody(w http.ResponseWriter, r *http.Request, target any) *Vomit {
	// First check if the body is of type JSON
	contentType := r.Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, contentJSON) {
		return puke(ReqIncorrectContentType, http.StatusUnsupportedMediaType, fmt.Sprintf("application/json Content-Type expected but got %s instead", contentType), nil)
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // Enable strict mode

	if err := dec.Decode(target); err != nil {
		// 1. Check for Body Too Large (Type check)
		var mbErr *http.MaxBytesError
		if errors.As(err, &mbErr) {
			return puke(ReqBodyTooLarge, http.StatusRequestEntityTooLarge, "body too large", err)
		}

		// 2. Check for Syntax Errors (Type check)
		var synErr *json.SyntaxError
		if errors.As(err, &synErr) {
			return puke(ReqJSONSyntaxError, http.StatusBadRequest, "syntax error in JSON", err)
		}

		// 3. Check for Type Mismatches (Type check)
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			return puke(ReqMalformedJSON, http.StatusBadRequest, "incorrect data type for field "+typeErr.Field, err)
		}

		// 4. Check for Unknown Fields (String check - no specific type exists!)
		if strings.HasPrefix(err.Error(), "json: unknown field") {
			return puke(ReqMalformedJSON, http.StatusBadRequest, err.Error(), err)
		}

		// 5. Check for Empty Body (Value check - using errors.Is)
		if errors.Is(err, io.EOF) {
			return puke(ReqEmptyBody, http.StatusBadRequest, "the request body is empty", err)
		}

		// 6. Check for Incomplete Payload.
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return puke(ReqMalformedJSON, http.StatusBadRequest, "the request payload stream is interrupted or the body is incomplete", err)
		}

		// 7. Generic Internal Error
		return puke(FlavorInternal, http.StatusInternalServerError, "internal error", err)
	}

	// check for the garbage leftover in the request
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return puke(ReqMalformedJSON, http.StatusBadRequest, "request body must only contain a single JSON object", err)
	}

	return nil
}

func respondWithJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", contentJSON)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// Handle potential encoding errors
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
