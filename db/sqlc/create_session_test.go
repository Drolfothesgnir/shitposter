package db

import (
	"context"
	"testing"
	"time"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Happy path: session is created and returned correctly.
func TestCreateSession_Success(t *testing.T) {
	ctx := context.Background()

	user := createRandomUser(t)

	arg := CreateSessionParams{
		ID:           uuid.New(),
		UserID:       user.ID,
		RefreshToken: util.RandomString(32),
		UserAgent:    "Firefox",
		ClientIp:     "192.168.1.1",
		IsBlocked:    false,
		ExpiresAt:    time.Now().Add(time.Minute),
	}

	session, err := testStore.CreateSession(ctx, arg)
	require.NoError(t, err)

	require.Equal(t, arg.ID, session.ID)
	require.Equal(t, arg.UserID, session.UserID)
	require.Equal(t, arg.RefreshToken, session.RefreshToken)
	require.Equal(t, arg.UserAgent, session.UserAgent)
	require.Equal(t, arg.ClientIp, session.ClientIp)
	require.Equal(t, arg.IsBlocked, session.IsBlocked)
	require.WithinDuration(t, arg.ExpiresAt, session.ExpiresAt, time.Millisecond)
	require.NotZero(t, session.CreatedAt)
}

// Non-existing user: should return OpError wrapping foreign key violation.
func TestCreateSession_UserNotFound(t *testing.T) {
	ctx := context.Background()

	arg := CreateSessionParams{
		ID:           uuid.New(),
		UserID:       999999,
		RefreshToken: util.RandomString(32),
		UserAgent:    "Chrome",
		ClientIp:     "10.0.0.1",
		IsBlocked:    false,
		ExpiresAt:    time.Now().Add(time.Minute),
	}

	session, err := testStore.CreateSession(ctx, arg)
	require.Error(t, err)

	require.Equal(t, uuid.Nil, session.ID)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opCreateSession, opErr.Op)
	require.Equal(t, entSession, opErr.Entity)
}

// Duplicate session ID: should return OpError.
func TestCreateSession_DuplicateID(t *testing.T) {
	ctx := context.Background()

	user := createRandomUser(t)

	arg := CreateSessionParams{
		ID:           uuid.New(),
		UserID:       user.ID,
		RefreshToken: util.RandomString(32),
		UserAgent:    "Safari",
		ClientIp:     "172.16.0.1",
		IsBlocked:    false,
		ExpiresAt:    time.Now().Add(time.Minute),
	}

	// first creation succeeds
	_, err := testStore.CreateSession(ctx, arg)
	require.NoError(t, err)

	// second creation with same ID fails
	arg.RefreshToken = util.RandomString(32)
	session, err := testStore.CreateSession(ctx, arg)
	require.Error(t, err)

	require.Equal(t, uuid.Nil, session.ID)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opCreateSession, opErr.Op)
	require.Equal(t, entSession, opErr.Entity)
}
