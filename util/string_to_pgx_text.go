package util

import (
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// StringToPgxText wraps pointer string to the pgtype.Text and trims the string if it is valid.
func StringToPgxText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}

	trim := strings.TrimSpace(*s)
	return pgtype.Text{String: trim, Valid: true}
}
