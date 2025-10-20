package db

import (
	"context"
	"testing"
	"time"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomUser(t *testing.T) User {

	arg := CreateUserParams{
		Username:           util.RandomOwner(),
		ProfileImgUrl:      pgtype.Text{String: util.RandomURL(), Valid: true},
		Email:              util.RandomEmail(),
		WebauthnUserHandle: util.RandomByteArray(32),
	}

	user, err := testStore.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Username, user.Username)
	require.Equal(t, user.Username, user.DisplayName)
	require.Equal(t, arg.ProfileImgUrl, user.ProfileImgUrl)
	require.Equal(t, arg.Email, user.Email)
	require.NotZero(t, user.CreatedAt)

	return user
}

func TestCreateUser(t *testing.T) {
	createRandomUser(t)
}

func TestSoftUserDelete(t *testing.T) {
	user1 := createRandomUser(t)

	user2, err := testStore.DeleteUser(context.Background(), user1.ID)
	require.NoError(t, err)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, "[deleted]", user2.DisplayName)
	require.True(t, user2.IsDeleted)
	require.WithinDuration(t, time.Now(), user2.DeletedAt, time.Minute)
}

// Test function used to check if you can create user
// with same email and username as some deleted user
func TestAvailableCredsAfterDelete(t *testing.T) {
	arg1 := CreateUserParams{
		Username:           util.RandomOwner(),
		Email:              util.RandomEmail(),
		WebauthnUserHandle: util.RandomByteArray(32),
	}

	user1, err := testStore.CreateUser(context.Background(), arg1)
	require.NoError(t, err)

	_, err = testStore.DeleteUser(context.Background(), user1.ID)
	require.NoError(t, err)

	arg2 := CreateUserParams{
		Username:           arg1.Username,
		Email:              arg1.Email,
		WebauthnUserHandle: util.RandomByteArray(32),
	}

	user3, err := testStore.CreateUser(context.Background(), arg2)
	require.NoError(t, err)

	require.Equal(t, arg1.Username, user3.Username)
	require.Equal(t, arg1.Email, user3.Email)

	arg3 := CreateUserParams{
		Username:           arg1.Username,
		Email:              arg1.Email,
		WebauthnUserHandle: util.RandomByteArray(32),
	}

	_, err = testStore.CreateUser(context.Background(), arg3)
	require.Error(t, err)

	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "23505", pgErr.Code)
}
