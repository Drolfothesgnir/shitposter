package db

import (
	"context"
	"testing"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/stretchr/testify/require"
)

func TestUsernameExists_TrueForExistingUser(t *testing.T) {
	ctx := context.Background()

	u := createRandomUser(t)

	exists, err := testStore.UsernameExists(ctx, u.Username)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestUsernameExists_FalseForRandomUsername(t *testing.T) {
	ctx := context.Background()

	randomUsername := "non_existing_" + util.RandomOwner()

	exists, err := testStore.UsernameExists(ctx, randomUsername)
	require.NoError(t, err)
	require.False(t, exists)
}

func TestEmailExists_TrueForExistingUser(t *testing.T) {
	ctx := context.Background()

	u := createRandomUser(t)

	exists, err := testStore.EmailExists(ctx, u.Email)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestEmailExists_FalseForRandomEmail(t *testing.T) {
	ctx := context.Background()

	randomEmail := "non-existing-" + util.RandomEmail()

	exists, err := testStore.EmailExists(ctx, randomEmail)
	require.NoError(t, err)
	require.False(t, exists)
}

// Soft-deleted user should free original username/email for reuse.
func TestUsernameAndEmailExists_SoftDeletedUserFreesValues(t *testing.T) {
	ctx := context.Background()

	u := createRandomUser(t)

	// Sanity: values are initially taken.
	exists, err := testStore.UsernameExists(ctx, u.Username)
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = testStore.EmailExists(ctx, u.Email)
	require.NoError(t, err)
	require.True(t, exists)

	// Soft-delete this user.
	_, err = testStore.SoftDeleteUserTx(ctx, u.ID)
	require.NoError(t, err)

	// After soft delete, the original username/email should no longer be present
	// in the users table, because your soft_delete_user function replaces them
	// with anonymized values and archives the old ones.
	exists, err = testStore.UsernameExists(ctx, u.Username)
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = testStore.EmailExists(ctx, u.Email)
	require.NoError(t, err)
	require.False(t, exists)
}
