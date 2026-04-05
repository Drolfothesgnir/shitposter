package api

import (
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStrURL(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		issues := []Issue{}

		proceed := strURL("https://example.com/avatar.png", "profile_img_url", &issues)

		require.True(t, proceed)
		require.Empty(t, issues)
	})

	t.Run("Invalid", func(t *testing.T) {
		issues := []Issue{}

		proceed := strURL("not a url", "profile_img_url", &issues)

		require.True(t, proceed)
		require.Len(t, issues, 1)
		require.Equal(t, Issue{
			FieldName: "profile_img_url",
			Tag:       validatorURL,
			Message:   "the value must be a valid URL",
		}, issues[0])
	})
}

func TestStrMin(t *testing.T) {
	t.Run("ValidAfterTrimmingUnicodeWhitespace", func(t *testing.T) {
		issues := []Issue{}

		proceed := strMin(3)("  абв  ", "username", &issues)

		require.True(t, proceed)
		require.Empty(t, issues)
	})

	t.Run("Invalid", func(t *testing.T) {
		issues := []Issue{}

		proceed := strMin(3)(" x ", "username", &issues)

		require.True(t, proceed)
		require.Len(t, issues, 1)
		require.Equal(t, Issue{
			FieldName: "username",
			Tag:       validatorMin,
			Message:   "value must be at least 3 symbols long, not including whitespaces",
		}, issues[0])
	})
}

func TestNumMinMax(t *testing.T) {
	t.Run("NumMin", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			issues := []Issue{}

			proceed := numMin(int32(5))(5, "root_offset", &issues)

			require.True(t, proceed)
			require.Empty(t, issues)
		})

		t.Run("Invalid", func(t *testing.T) {
			issues := []Issue{}

			proceed := numMin(int32(5))(4, "root_offset", &issues)

			require.True(t, proceed)
			require.Len(t, issues, 1)
			require.Equal(t, Issue{
				FieldName: "root_offset",
				Tag:       validatorMin,
				Message:   "value must be at least 5",
			}, issues[0])
		})
	})

	t.Run("NumMax", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			issues := []Issue{}

			proceed := numMax(int32(10))(10, "n_roots", &issues)

			require.True(t, proceed)
			require.Empty(t, issues)
		})

		t.Run("Invalid", func(t *testing.T) {
			issues := []Issue{}

			proceed := numMax(int32(10))(11, "n_roots", &issues)

			require.True(t, proceed)
			require.Len(t, issues, 1)
			require.Equal(t, Issue{
				FieldName: "n_roots",
				Tag:       validatorMax,
				Message:   "value must be at most 10",
			}, issues[0])
		})
	})
}

func TestExtractRequiredParam(t *testing.T) {
	t.Run("MissingParam", func(t *testing.T) {
		issues := []Issue{}
		values := url.Values{}
		dest := int32(99)

		extractRequiredParam(&issues, values, "root_offset", &dest, parseSingle(parseInt32))

		require.Equal(t, int32(99), dest)
		require.Len(t, issues, 1)
		require.Equal(t, Issue{
			FieldName: "root_offset",
			Tag:       validatorRequired,
			Message:   "root_offset must be provided",
		}, issues[0])
	})

	t.Run("ValidParam", func(t *testing.T) {
		issues := []Issue{}
		values := url.Values{"root_offset": []string{"42"}}
		dest := int32(0)

		extractRequiredParam(&issues, values, "root_offset", &dest, parseSingle(parseInt32), numMin(int32(0)))

		require.Empty(t, issues)
		require.Equal(t, int32(42), dest)
	})
}

func TestParseSingle(t *testing.T) {
	t.Run("SingleValue", func(t *testing.T) {
		parse := parseSingle(parseInt32)

		got, err := parse([]string{"17"})

		require.NoError(t, err)
		require.Equal(t, int32(17), got)
	})

	t.Run("MultipleValues", func(t *testing.T) {
		parse := parseSingle(parseString)

		got, err := parse([]string{"old", "new"})

		require.Error(t, err)
		require.Empty(t, got)
		require.EqualError(t, err, `there should be only one value for this field, received: "old", "new"`)
	})
}

func TestParseAndValidate(t *testing.T) {
	t.Run("ParseError", func(t *testing.T) {
		issues := []Issue{}
		dest := int32(55)

		parse := func([]string) (int32, error) {
			return 0, errors.New("bad integer")
		}

		parseAndValidate(&issues, []string{"abc"}, "root_offset", &dest, parse, nil)

		require.Equal(t, int32(55), dest)
		require.Len(t, issues, 1)
		require.Equal(t, Issue{
			FieldName: "root_offset",
			Tag:       "type_error",
			Message:   "root_offset has an invalid format: bad integer",
		}, issues[0])
	})

	t.Run("StopsValidationChainAfterFalse", func(t *testing.T) {
		issues := []Issue{}
		dest := ""
		secondCalled := false

		first := func(v string, fieldName string, issues *[]Issue) bool {
			*issues = append(*issues, Issue{
				FieldName: fieldName,
				Tag:       "first",
				Message:   fmt.Sprintf("saw %q", v),
			})
			return false
		}

		second := func(v string, fieldName string, issues *[]Issue) bool {
			secondCalled = true
			return true
		}

		parseAndValidate(&issues, []string{"pop"}, "order", &dest, parseSingle(parseString), []validator[string]{first, second})

		require.False(t, secondCalled)
		require.Equal(t, "pop", dest)
		require.Len(t, issues, 1)
		require.Equal(t, Issue{
			FieldName: "order",
			Tag:       "first",
			Message:   `saw "pop"`,
		}, issues[0])
	})
}
