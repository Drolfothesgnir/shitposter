package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSoftDeleteUserTx(t *testing.T) {
	ctx := context.Background()

	// 1️⃣ — Creating test user
	user := createRandomUser(t)

	// 2️⃣ — Creating mock data
	_ = createSessionForUser(t, user.ID)

	_ = createCredentialForUser(t, user.ID)

	// Check if data exist
	gotUser, err := testStore.GetUser(ctx, user.ID)
	require.NoError(t, err)
	require.False(t, gotUser.IsDeleted)
	require.Equal(t, user.Username, gotUser.Username)

	// 3️⃣ — Calling SoftDeleteUserTx
	err = testStore.SoftDeleteUserTx(ctx, user.ID)
	require.NoError(t, err)

	// 4️⃣ — Check if sessions and creds have been deleted
	sessions, err := testStore.ListSessionsByUser(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, sessions, 0, "sessions should be deleted")

	creds, err := testStore.ListUserCredentials(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, creds, 0, "webauthn credentials should be deleted")

	// 5️⃣ — Check if user has been marked as deleted
	deletedUser, err := testStore.GetUser(ctx, user.ID)
	require.NoError(t, err)
	require.True(t, deletedUser.IsDeleted, "user must be marked deleted")
	require.WithinDuration(t, time.Now(), deletedUser.DeletedAt, 2*time.Second)
	require.Contains(t, deletedUser.Username, "deleted_user_")
	require.Contains(t, deletedUser.Email, "@invalid.local")

	// 6️⃣ — Repeat call should be idempotent
	err = testStore.SoftDeleteUserTx(ctx, user.ID)
	require.NoError(t, err, "repeated soft-delete should not fail")
}
