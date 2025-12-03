package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// Happy path: user exists and is not soft-deleted.
func TestGetUser_Success(t *testing.T) {
	ctx := context.Background()

	// createRandomUser already uses testStore internally
	u := createRandomUser(t)

	got, err := testStore.GetUser(ctx, u.ID)
	require.NoError(t, err)

	// basic identity checks
	require.EqualValues(t, u.ID, got.ID)
	require.Equal(t, u.Username, got.Username)
	require.Equal(t, u.Email, got.Email)

	// should not be marked as deleted
	require.False(t, got.IsDeleted)
}

// Non-existing user: should return OpError with KindNotFound.
func TestGetUser_NotFound(t *testing.T) {
	ctx := context.Background()

	nonExistingID := int64(9_999_999_999)

	u, err := testStore.GetUser(ctx, nonExistingID)
	require.Error(t, err)

	// returned user must be zero value
	require.Zero(t, u.ID)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opGetUser, opErr.Op)
	require.Equal(t, KindNotFound, opErr.Kind)
	require.Equal(t, entUser, opErr.Entity)
	require.EqualValues(t, nonExistingID, opErr.EntityID)
}

// Soft-deleted user: GetUser should return KindDeleted.
func TestGetUser_DeletedUser(t *testing.T) {
	ctx := context.Background()

	// create a normal user first
	u := createRandomUser(t)

	// soft-delete the user
	deleted, err := testStore.softDeleteUser(ctx, u.ID)
	require.NoError(t, err)
	require.True(t, deleted.IsDeleted)

	// now GetUser should treat this as "deleted"
	got, err := testStore.GetUser(ctx, u.ID)
	require.Error(t, err)

	// result should be zero value
	require.Zero(t, got.ID)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opGetUser, opErr.Op)
	require.Equal(t, KindDeleted, opErr.Kind)
	require.Equal(t, entUser, opErr.Entity)
	require.EqualValues(t, u.ID, opErr.EntityID)
}
