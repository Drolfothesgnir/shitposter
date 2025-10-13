package db

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func randomUserHandle(t *testing.T) []byte {
	handle := make([]byte, 32)
	_, err := rand.Read(handle)
	require.NoError(t, err)
	return handle
}

func createRandomUser(t *testing.T) User {

	arg := CreateUserParams{
		Username:           util.RandomOwner(),
		ProfileImgUrl:      pgtype.Text{String: util.RandomURL(), Valid: true},
		Email:              util.RandomEmail(),
		WebauthnUserHandle: randomUserHandle(t),
	}

	user, err := testStore.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Username, user.Username)
	require.Equal(t, arg.ProfileImgUrl, user.ProfileImgUrl)
	require.Equal(t, arg.Email, user.Email)
	require.NotZero(t, user.CreatedAt)

	return user
}

func TestCreateUser(t *testing.T) {
	createRandomUser(t)
}
