package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Happy path: session exists and is returned correctly.
func TestGetSession_Success(t *testing.T) {
	ctx := context.Background()

	user := createRandomUser(t)
	session := createSessionForUser(t, user.ID)

	got, err := testStore.GetSession(ctx, session.ID)
	require.NoError(t, err)

	require.Equal(t, session.ID, got.ID)
	require.Equal(t, session.UserID, got.UserID)
	require.Equal(t, session.RefreshToken, got.RefreshToken)
	require.Equal(t, session.UserAgent, got.UserAgent)
	require.Equal(t, session.ClientIp, got.ClientIp)
	require.Equal(t, session.IsBlocked, got.IsBlocked)
	require.WithinDuration(t, session.ExpiresAt, got.ExpiresAt, 0)
	require.WithinDuration(t, session.CreatedAt, got.CreatedAt, 0)
}

// Non-existing session: should return OpError with KindNotFound.
func TestGetSession_NotFound(t *testing.T) {
	ctx := context.Background()

	nonExistingID := uuid.New()

	got, err := testStore.GetSession(ctx, nonExistingID)
	require.Error(t, err)

	require.Equal(t, uuid.Nil, got.ID)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opGetSession, opErr.Op)
	require.Equal(t, KindNotFound, opErr.Kind)
	require.Equal(t, entSession, opErr.Entity)
	require.Equal(t, nonExistingID.String(), opErr.EntityID)
}
