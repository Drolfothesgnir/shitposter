package db

import (
	"context"
	"testing"
	"time"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/stretchr/testify/require"
)

// helper to get *string
func strPtr(s string) *string { return &s }

// Happy path: update a subset of fields (e.g. username only).
func TestUpdateUser_Success_Partial(t *testing.T) {
	ctx := context.Background()

	u := createRandomUser(t)

	newUsername := "updated_" + util.RandomOwner()

	arg := UpdateUserParams{
		ID:       u.ID,
		Username: &newUsername,
		// Email and ProfileImgURL left nil -> not updated
	}

	before := time.Now()

	res, err := testStore.UpdateUser(ctx, arg)
	require.NoError(t, err)

	require.EqualValues(t, u.ID, res.ID)
	require.Equal(t, newUsername, res.Username)
	require.Equal(t, u.Email, res.Email)                        // not changed
	require.Equal(t, u.ProfileImgUrl.String, res.ProfileImgURL) // not changed
	require.False(t, res.LastModifiedAt.Before(before))

	// Double-check DB state
	reloaded, err := testStore.getUser(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, newUsername, reloaded.Username)
	require.Equal(t, u.Email, reloaded.Email)
	require.Equal(t, u.ProfileImgUrl, reloaded.ProfileImgUrl)
	require.False(t, reloaded.LastModifiedAt.Before(before))
}

// All optional fields nil -> KindInvalid.
func TestUpdateUser_AllFieldsEmpty(t *testing.T) {
	ctx := context.Background()

	u := createRandomUser(t)

	arg := UpdateUserParams{
		ID:            u.ID,
		Username:      nil,
		Email:         nil,
		ProfileImgURL: nil,
	}

	_, err := testStore.UpdateUser(ctx, arg)
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opUpdateUser, opErr.Op)
	require.Equal(t, KindInvalid, opErr.Kind)
	require.Equal(t, entUser, opErr.Entity)
}

// Non-existing user -> KindNotFound.
func TestUpdateUser_NotFound(t *testing.T) {
	ctx := context.Background()

	nonExistingID := int64(9_999_999_999)

	newEmail := "new_" + util.RandomEmail()

	arg := UpdateUserParams{
		ID:    nonExistingID,
		Email: &newEmail,
	}

	_, err := testStore.UpdateUser(ctx, arg)
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opUpdateUser, opErr.Op)
	require.Equal(t, KindNotFound, opErr.Kind)
	require.Equal(t, entUser, opErr.Entity)
	require.EqualValues(t, nonExistingID, opErr.EntityID)
}

// Soft-deleted user cannot be updated -> KindDeleted.
func TestUpdateUser_DeletedUser(t *testing.T) {
	ctx := context.Background()

	u := createRandomUser(t)

	// Soft-delete the user
	deleted, err := testStore.softDeleteUser(ctx, u.ID)
	require.NoError(t, err)
	require.True(t, deleted.IsDeleted)

	newUsername := "updated_" + util.RandomOwner()

	arg := UpdateUserParams{
		ID:       u.ID,
		Username: &newUsername,
	}

	_, err = testStore.UpdateUser(ctx, arg)
	require.Error(t, err)
	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opUpdateUser, opErr.Op)
	require.Equal(t, KindDeleted, opErr.Kind)
	require.Equal(t, entUser, opErr.Entity)
	require.EqualValues(t, u.ID, opErr.EntityID)

	// Ensure nothing was changed in DB
	_, err = testStore.GetUser(ctx, u.ID)
	require.Error(t, err) // GetUser should itself return KindDeleted
}

// Email conflict: try to set email to one that already belongs to another user.
// Should be mapped by sqlError to KindConflict.
func TestUpdateUser_EmailConflict(t *testing.T) {
	ctx := context.Background()

	// user1 with email1
	user1 := createRandomUser(t)
	// user2 with email2
	user2 := createRandomUser(t)

	// try to set user2.Email = user1.Email -> unique violation
	arg := UpdateUserParams{
		ID:    user2.ID,
		Email: strPtr(user1.Email),
	}

	_, err := testStore.UpdateUser(ctx, arg)
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opUpdateUser, opErr.Op)
	require.Equal(t, KindConflict, opErr.Kind)
	require.Equal(t, entUser, opErr.Entity)

	// user2 must still have old email
	reloaded2, err := testStore.getUser(ctx, user2.ID)
	require.NoError(t, err)
	require.Equal(t, user2.Email, reloaded2.Email)
}
