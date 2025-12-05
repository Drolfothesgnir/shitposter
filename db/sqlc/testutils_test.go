package db

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomComment(t *testing.T) Comment {
	t.Helper()

	ctx := context.Background()
	post := createRandomPost(t)

	arg := createCommentParams{
		UserID: post.UserID,
		PostID: post.ID,
		Body:   util.RandomString(10),
	}

	comment, err := testStore.createComment(ctx, arg)
	require.NoError(t, err)

	require.Equal(t, arg.UserID, comment.UserID)
	require.Equal(t, arg.PostID, comment.PostID)
	require.Equal(t, arg.Body, comment.Body)
	require.Equal(t, int32(0), comment.Depth)
	require.False(t, comment.ParentID.Valid)
	require.Zero(t, comment.Downvotes)
	require.Zero(t, comment.Upvotes)

	return comment
}

func TestCreateComment(t *testing.T) {
	createRandomComment(t)
}

func createRandomUser(t *testing.T) User {

	arg := createUserParams{
		Username:           util.RandomOwner(),
		ProfileImgUrl:      pgtype.Text{String: util.RandomURL(), Valid: true},
		Email:              util.RandomEmail(),
		WebauthnUserHandle: util.RandomByteArray(32),
	}

	user, err := testStore.createUser(context.Background(), arg)
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

func credentialParamsTx() CreateCredentialsTxParams {

	aaguid := util.RandomByteArray(16)

	transports := []protocol.AuthenticatorTransport{
		protocol.USB,
		protocol.NFC,
	}

	transportsJson, _ := json.Marshal(transports)

	arg := CreateCredentialsTxParams{
		ID:                      util.RandomByteArray(32),
		PublicKey:               util.RandomByteArray(32),
		Transports:              transportsJson,
		UserPresent:             true,
		UserVerified:            true,
		BackupEligible:          true,
		BackupState:             true,
		Aaguid:                  uuid.UUID(aaguid),
		CloneWarning:            false,
		AuthenticatorAttachment: AuthenticatorAttachment(protocol.Platform),
		AuthenticatorData:       []byte{},
		PublicKeyAlgorithm:      -7,
	}

	return arg
}

func credentialParams(t *testing.T, userID int64) createWebauthnCredentialsParams {
	aaguid := util.RandomByteArray(16)

	transports := []protocol.AuthenticatorTransport{
		protocol.USB,
		protocol.NFC,
	}

	transportsJson, err := json.Marshal(transports)
	require.NoError(t, err)

	arg := createWebauthnCredentialsParams{
		ID:                      util.RandomByteArray(32),
		UserID:                  userID,
		PublicKey:               util.RandomByteArray(32),
		Transports:              transportsJson,
		UserPresent:             true,
		UserVerified:            true,
		BackupEligible:          true,
		BackupState:             true,
		Aaguid:                  uuid.UUID(aaguid),
		CloneWarning:            false,
		AuthenticatorAttachment: AuthenticatorAttachment(protocol.Platform),
		AuthenticatorData:       []byte{},
		PublicKeyAlgorithm:      -7,
	}

	return arg
}

func createCredentialForUser(t *testing.T, userID int64) WebauthnCredential {

	arg := credentialParams(t, userID)
	cred, err := testStore.createWebauthnCredentials(context.Background(), arg)
	require.NoError(t, err)

	return cred
}

// Creates a credential for a user with a specific ID (used to force conflicts).
func createCredentialForUserWithID(t *testing.T, userID int64, id []byte) WebauthnCredential {
	t.Helper()

	arg := credentialParams(t, userID)
	arg.ID = id // override random ID with a fixed one

	cred, err := testStore.createWebauthnCredentials(context.Background(), arg)
	require.NoError(t, err)

	return cred
}
