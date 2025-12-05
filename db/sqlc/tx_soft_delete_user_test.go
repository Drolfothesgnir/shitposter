package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Soft delete happy path: active user gets anonymized and marked deleted.
func TestSoftDeleteUserTx_Success(t *testing.T) {
	ctx := context.Background()

	// Create an active user.
	u := createRandomUser(t)

	before := time.Now()

	res, err := testStore.SoftDeleteUserTx(ctx, u.ID)
	require.NoError(t, err)

	// Basic identity
	require.EqualValues(t, u.ID, res.ID)
	require.True(t, res.IsDeleted)

	// Public-facing fields should be scrubbed.
	require.Equal(t, "[deleted]", res.DisplayName)
	require.Equal(t, fmt.Sprintf("deleted_user_%d", u.ID), res.Username)
	require.Equal(t, fmt.Sprintf("deleted_%d@invalid.local", u.ID), res.Email)
	t.Log(res.ProfileImgURL)
	// Profile image URL must be cleared.
	require.False(t, res.ProfileImgURL.Valid)

	// Deletion timestamps must be set.
	require.False(t, res.DeletedAt.IsZero())
	require.False(t, res.LastModifiedAt.IsZero())
	require.False(t, res.DeletedAt.Before(before))
	require.False(t, res.LastModifiedAt.Before(before))

	// Subsequent GetUser should report a deleted user.
	_, err = testStore.GetUser(ctx, u.ID)
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opGetUser, opErr.Op)
	require.Equal(t, KindDeleted, opErr.Kind)
	require.Equal(t, entUser, opErr.Entity)
	require.EqualValues(t, u.ID, opErr.EntityID)
}

// Non-existing user: SoftDeleteUserTx should return KindNotFound.
func TestSoftDeleteUserTx_NotFound(t *testing.T) {
	ctx := context.Background()

	nonExistingID := int64(9_999_999_999)

	_, err := testStore.SoftDeleteUserTx(ctx, nonExistingID)
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opSoftDeleteUser, opErr.Op)
	require.Equal(t, KindNotFound, opErr.Kind)
	require.Equal(t, entUser, opErr.Entity)
	require.EqualValues(t, nonExistingID, opErr.EntityID)
}

// Idempotency: calling SoftDeleteUserTx twice should still succeed and keep user deleted.
func TestSoftDeleteUserTx_Idempotent(t *testing.T) {
	ctx := context.Background()

	u := createRandomUser(t)

	// First delete.
	first, err := testStore.SoftDeleteUserTx(ctx, u.ID)
	require.NoError(t, err)
	require.True(t, first.IsDeleted)

	// Second delete on already-deleted user.
	second, err := testStore.SoftDeleteUserTx(ctx, u.ID)
	require.NoError(t, err)
	require.True(t, second.IsDeleted)

	// The anonymized fields should remain consistent.
	require.Equal(t, first.DisplayName, second.DisplayName)
	require.Equal(t, first.Username, second.Username)
	require.Equal(t, first.Email, second.Email)

	// Profile image remains cleared.
	require.False(t, second.ProfileImgURL.Valid)

	// Timestamps should still be non-zero.
	require.False(t, second.DeletedAt.IsZero())
	require.False(t, second.LastModifiedAt.IsZero())
}
