package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

// local validator, for Field() in ValidationErrors to return json-name
func newValidatorJSON() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return v
}

func TestExtractErrorFields_NonValidationError(t *testing.T) {
	fields := ExtractErrorFields(errors.New("not a validation error"))
	require.Empty(t, fields)
}

// helper: validate struct, get single error and check fields
func checkSingleFieldError(t *testing.T, v *validator.Validate, s any, expectedField, expectedMsg string) {
	t.Helper()
	err := v.Struct(s)
	require.Error(t, err)

	fields := ExtractErrorFields(err)
	require.Len(t, fields, 1)
	require.Equal(t, expectedField, fields[0].FieldName)
	require.Equal(t, expectedMsg, fields[0].ErrorMessage)
}

func TestExtractErrorFields_TagMessages(t *testing.T) {
	v := newValidatorJSON()

	t.Run("required", func(t *testing.T) {
		type S struct {
			Username string `json:"username" validate:"required"`
		}
		checkSingleFieldError(t, v, S{Username: ""}, "username", "this field is required")
	})

	t.Run("min", func(t *testing.T) {
		type S struct {
			Username string `json:"username" validate:"min=3"`
		}
		checkSingleFieldError(t, v, S{Username: "ab"}, "username", "value is too short")
	})

	t.Run("max", func(t *testing.T) {
		type S struct {
			Username string `json:"username" validate:"max=2"`
		}
		checkSingleFieldError(t, v, S{Username: "abc"}, "username", "value is too long")
	})

	t.Run("len", func(t *testing.T) {
		type S struct {
			Code string `json:"code" validate:"len=5"`
		}
		checkSingleFieldError(t, v, S{Code: "abcd"}, "code", "invalid length")
	})

	t.Run("email", func(t *testing.T) {
		type S struct {
			Email string `json:"email" validate:"email"`
		}
		checkSingleFieldError(t, v, S{Email: "not-an-email"}, "email", "invalid email address")
	})

	t.Run("url", func(t *testing.T) {
		type S struct {
			Link string `json:"link" validate:"url"`
		}
		// Pick ONE value below:
		// s := S{Link: "http//bad"}        // missing colon after scheme
		// s := S{Link: "http:/bad"}        // single slash
		// s := S{Link: "://no-scheme"}     // no scheme
		// s := S{Link: "example.com"}      // no scheme at all
		// s := S{Link: "http://"}          // empty host
		s := S{Link: "http//bad"}

		checkSingleFieldError(t, v, s, "link", "invalid URL format")
	})

	t.Run("alphanum", func(t *testing.T) {
		type S struct {
			Username string `json:"username" validate:"alphanum"`
		}
		checkSingleFieldError(t, v, S{Username: "abc-123"}, "username", "must contain only letters and numbers")
	})

	t.Run("alpha", func(t *testing.T) {
		type S struct {
			Name string `json:"name" validate:"alpha"`
		}
		checkSingleFieldError(t, v, S{Name: "John3"}, "name", "must contain only letters")
	})

	t.Run("numeric", func(t *testing.T) {
		type S struct {
			Code string `json:"code" validate:"numeric"`
		}
		checkSingleFieldError(t, v, S{Code: "12a"}, "code", "must contain only numbers")
	})

	t.Run("gte", func(t *testing.T) {
		type S struct {
			Age int `json:"age" validate:"gte=18"`
		}
		checkSingleFieldError(t, v, S{Age: 16}, "age", "must be greater than or equal to the allowed minimum")
	})

	t.Run("lte", func(t *testing.T) {
		type S struct {
			Age int `json:"age" validate:"lte=10"`
		}
		checkSingleFieldError(t, v, S{Age: 11}, "age", "must be less than or equal to the allowed maximum")
	})

	t.Run("gt", func(t *testing.T) {
		type S struct {
			N int `json:"n" validate:"gt=5"`
		}
		checkSingleFieldError(t, v, S{N: 5}, "n", "must be greater than the allowed minimum")
	})

	t.Run("lt", func(t *testing.T) {
		type S struct {
			N int `json:"n" validate:"lt=5"`
		}
		checkSingleFieldError(t, v, S{N: 5}, "n", "must be less than the allowed maximum")
	})

	t.Run("oneof", func(t *testing.T) {
		type S struct {
			Color string `json:"color" validate:"oneof=red blue"`
		}
		checkSingleFieldError(t, v, S{Color: "green"}, "color", "must be one of the allowed values")
	})

	t.Run("uuid", func(t *testing.T) {
		type S struct {
			ID string `json:"id" validate:"uuid"`
		}
		checkSingleFieldError(t, v, S{ID: "not-a-uuid"}, "id", "invalid UUID format")
	})

	t.Run("ip", func(t *testing.T) {
		type S struct {
			Addr string `json:"addr" validate:"ip"`
		}
		checkSingleFieldError(t, v, S{Addr: "999.999.999.999"}, "addr", "invalid IP address")
	})

	t.Run("ipv4", func(t *testing.T) {
		type S struct {
			Addr string `json:"addr" validate:"ipv4"`
		}
		checkSingleFieldError(t, v, S{Addr: "abcd"}, "addr", "invalid IPv4 address")
	})

	t.Run("ipv6", func(t *testing.T) {
		type S struct {
			Addr string `json:"addr" validate:"ipv6"`
		}
		checkSingleFieldError(t, v, S{Addr: "1234"}, "addr", "invalid IPv6 address")
	})

	t.Run("startswith", func(t *testing.T) {
		type S struct {
			Slug string `json:"slug" validate:"startswith=foo"`
		}
		checkSingleFieldError(t, v, S{Slug: "barbaz"}, "slug", "must start with the required prefix")
	})

	t.Run("endswith", func(t *testing.T) {
		type S struct {
			Slug string `json:"slug" validate:"endswith=foo"`
		}
		checkSingleFieldError(t, v, S{Slug: "bar"}, "slug", "must end with the required suffix")
	})

	t.Run("default_fallback_for_unknown_but_valid_tag", func(t *testing.T) {
		// using valid tag, which is not in mapping, to get into default case.
		type S struct {
			Color string `json:"color" validate:"hexcolor"`
		}
		checkSingleFieldError(t, v, S{Color: "not-hex"}, "color", "invalid input")
	})
}

func TestExtractErrorFromBuffer(t *testing.T) {
	exp := ErrorResponse{
		Error: "invalid params",
		Fields: []ErrorField{
			{FieldName: "username", ErrorMessage: "must contain only letters and numbers"},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, json.NewEncoder(&buf).Encode(exp))

	got, err := extractErrorFromBuffer(&buf)
	require.NoError(t, err)
	require.Equal(t, exp, *got)
}
