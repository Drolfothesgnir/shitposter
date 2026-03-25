package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const opCreateSession = "create-session"

type CreateSessionParams struct {
	ID           uuid.UUID `json:"id"`
	UserID       int64     `json:"user_id"`
	RefreshToken string    `json:"refresh_token"`
	UserAgent    string    `json:"user_agent"`
	ClientIp     string    `json:"client_ip"`
	IsBlocked    bool      `json:"is_blocked"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// CreateSession creates user session with arguments provided in arg.
// Returns newly created [Session] and possible [OpError] with [KindInternal] in case of the database error.
func (s *SQLStore) CreateSession(ctx context.Context, arg CreateSessionParams) (Session, error) {
	session, err := s.createSession(ctx, createSessionParams{
		ID:           arg.ID,
		UserID:       arg.UserID,
		RefreshToken: arg.RefreshToken,
		UserAgent:    arg.UserAgent,
		ClientIp:     arg.ClientIp,
		IsBlocked:    arg.IsBlocked,
		ExpiresAt:    arg.ExpiresAt,
	})

	if err != nil {
		opErr := sqlError(
			opCreateSession,
			opDetails{
				entity:   entSession,
				entityID: arg.ID.String(),
				userID:   fmt.Sprint(arg.UserID),
			},
			err,
		)
		return Session{}, opErr
	}

	return session, nil
}
