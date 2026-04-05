// TODO: write some docs
package api

import (
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	validatorRequired = "required"
	validatorEmail    = "email"
	validatorURL      = "url"
	validatorMin      = "min"
	validatorMax      = "max"
	validatorAlphanum = "alphanum"
)

// barf makes a *Vomit out of list of particular field errors.
func barf(issues []Issue) *Vomit {
	if len(issues) > 0 {
		return puke(
			ReqInvalidArguments,
			http.StatusBadRequest,
			"invalid request arguments",
			nil, // what to put here?
			issues...,
		)
	}

	return nil
}

func strRequired(v, fieldName string, issues *[]Issue) bool {
	if strings.TrimSpace(v) == "" {
		*issues = append(*issues, Issue{
			FieldName: fieldName,
			Tag:       validatorRequired,
			Message:   "field is required",
		})

		return false
	}

	return true
}

func strEmail(v, fieldName string, issues *[]Issue) bool {
	if _, err := mail.ParseAddress(v); err != nil {
		*issues = append(*issues, Issue{
			FieldName: fieldName,
			Tag:       validatorEmail,
			Message:   "field must be a correct email address",
		})
	}

	return true
}

func strURL(v, fieldName string, issues *[]Issue) bool {
	if _, err := url.ParseRequestURI(v); err != nil {
		*issues = append(*issues, Issue{
			FieldName: fieldName,
			Tag:       validatorURL,
			Message:   "the value must be a valid URL",
		})
	}

	return true
}

func strMin(min int) validator[string] {
	return func(v, fieldName string, issues *[]Issue) bool {
		trimmed := strings.TrimSpace(v)

		// countring UTF-8 characters instead of bytes
		if utf8.RuneCountInString(trimmed) < min {
			*issues = append(*issues, Issue{
				FieldName: fieldName,
				Tag:       validatorMin,
				Message:   fmt.Sprintf("value must be at least %d symbols long, not including whitespaces", min),
			})
		}

		return true
	}
}

func strMax(max int) validator[string] {
	return func(v, fieldName string, issues *[]Issue) bool {
		trimmed := strings.TrimSpace(v)

		// countring UTF-8 characters instead of bytes
		if utf8.RuneCountInString(trimmed) > max {
			*issues = append(*issues, Issue{
				FieldName: fieldName,
				Tag:       validatorMax,
				Message:   fmt.Sprintf("value must be at most %d symbols long, not including whitespaces", max),
			})
		}

		return true
	}
}

func isAlphanumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) && !unicode.IsLetter(r) {
			return false
		}
	}

	return true
}

func strAlphanum(v, fieldName string, issues *[]Issue) bool {
	trimmed := strings.TrimSpace(v)

	if !isAlphanumeric(trimmed) {
		*issues = append(*issues, Issue{
			FieldName: fieldName,
			Tag:       validatorAlphanum,
			Message:   "value must only letters and numbers",
		})
	}

	return true
}

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// type number interface {
// 	integer | ~float32 | ~float64
// }

func numMin[T integer](min T) func(v T, fieldName string, issues *[]Issue) bool {
	return func(v T, fieldName string, issues *[]Issue) bool {
		if v < min {
			*issues = append(*issues, Issue{
				FieldName: fieldName,
				Tag:       validatorMin,
				Message:   fmt.Sprintf("value must be at least %d", min),
			})
		}

		return true
	}
}

func numMax[T integer](max T) func(v T, fieldName string, issues *[]Issue) bool {
	return func(v T, fieldName string, issues *[]Issue) bool {
		if v > max {
			*issues = append(*issues, Issue{
				FieldName: fieldName,
				Tag:       validatorMax,
				Message:   fmt.Sprintf("value must be at most %d", max),
			})
		}

		return true
	}
}

// validator is a function which performs neccessary checks on the argument v
// and append [Issue] to the issues slice if there are any.
// Returns false if no other validations should be performed on the value,
// so order matters when using in [validate] func!
type validator[T any] func(v T, fieldName string, issues *[]Issue) (proceed bool)

// validate performs chain of checks on value v of the field <fieldName> with validators from variadic fns
// and appends any [Issue] to the issues slice.
func validate[T any](issues *[]Issue, v T, fieldName string, fns ...validator[T]) {
	for _, fn := range fns {
		if proceed := fn(v, fieldName, issues); !proceed {
			break
		}
	}
}

// Parser defines a function that converts a string to any type T
type parser[T any] func([]string) (T, error)

type singleParser[T any] func(string) (T, error)

func extractRequiredParam[T any](
	issues *[]Issue,
	m url.Values,
	key string,
	dest *T,
	parse parser[T],
	fns ...validator[T],
) {
	// 1. Get the value (Extract)
	vals, ok := m[key]
	if !ok || len(vals) == 0 {
		*issues = append(*issues, Issue{
			FieldName: key,
			Tag:       validatorRequired,
			Message:   fmt.Sprintf("%s must be provided", key),
		})
		return
	}
	parseAndValidate(issues, vals, key, dest, parse, fns)
}

func extractOptionalParam[T any](
	issues *[]Issue,
	m url.Values,
	key string,
	dest *T,
	parse parser[T],
	fns ...validator[T],
) {
	vals, ok := m[key]
	if !ok || len(vals) == 0 {
		return
	}
	parseAndValidate(issues, vals, key, dest, parse, fns)
}

func parseSingle[T any](p singleParser[T]) parser[T] {
	return func(s []string) (T, error) {
		var zero T

		if len(s) > 1 {
			quoted := make([]string, len(s))
			for i, str := range s {
				quoted[i] = strconv.Quote(str)
			}

			joined := strings.Join(quoted, ", ")

			return zero, fmt.Errorf("there should be only one value for this field, received: %s", joined)
		}

		return p(s[0])
	}
}

func parseInt32(s string) (int32, error) {
	val, err := strconv.ParseInt(s, 10, 32)
	return int32(val), err
}

func parseString(s string) (string, error) {
	return s, nil // Strings just pass through
}

func parseAndValidate[T any](issues *[]Issue, vals []string, key string, dest *T, parse parser[T], fns []validator[T]) {
	parsed, err := parse(vals)
	if err != nil {
		*issues = append(*issues, Issue{
			FieldName: key,
			Tag:       "type_error",
			Message:   fmt.Sprintf("%s has an invalid format: %s", key, err.Error()),
		})
		return
	}

	for _, fn := range fns {
		if proceed := fn(parsed, key, issues); !proceed {
			break // Stop validating this field if a validator fails and says to stop
		}
	}

	*dest = parsed
}
