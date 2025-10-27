package util

import (
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

func StringToPgxText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}

	trim := strings.TrimSpace(*s)
	return pgtype.Text{String: trim, Valid: true}
}
