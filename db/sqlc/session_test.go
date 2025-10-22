package db

import (
	"context"
	"testing"
	"time"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createSessionForUser(t *testing.T, userID int64) Session {
	arg := CreateSessionParams{
		ID:           uuid.New(),
		UserID:       userID,
		RefreshToken: util.RandomString(32),
		UserAgent:    "Chrome",
		ClientIp:     "198.162.0.0",
		IsBlocked:    false,
		ExpiresAt:    time.Now().Add(time.Minute),
	}
	session, err := testStore.CreateSession(context.Background(), arg)
	require.NoError(t, err)

	return session
}
