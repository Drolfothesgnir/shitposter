package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const opGetSession = "get-session"

// GetSession retrieves the session with the provided ID.
// Returns KindNotFound if the session does not exist, or KindInternal on database errors.
func (s *SQLStore) GetSession(ctx context.Context, id uuid.UUID) (Session, error) {
	session, err := s.getSession(ctx, id)
	if err == pgx.ErrNoRows {
		return Session{}, notFoundError(opGetSession, entSession, id.String())
	}

	if err != nil {
		opErr := sqlError(
			opGetSession,
			opDetails{
				entity: entSession,
				input:  id.String(),
			},
			err)

		return Session{}, opErr
	}

	return session, nil
}
