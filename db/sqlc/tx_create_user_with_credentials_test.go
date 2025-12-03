package db

import (
	"context"
	"testing"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCreateUserWithCredentials_Success(t *testing.T) {
	ctx := context.Background()

	imgUrl := util.RandomURL()
	u := NewCreateUserParams(
		util.RandomOwner(),
		util.RandomEmail(),
		&imgUrl,
		util.RandomByteArray(10),
	)
	cred := credentialParamsTx()

	res, err := testStore.CreateUserWithCredentialsTx(ctx,
		CreateUserWithCredentialsTxParams{
			User: u,
			Cred: cred,
		})

	require.NoError(t, err)
	require.NotZero(t, res.User.ID)

	// credential exists
	c, err := testStore.getUserCredentials(ctx, res.User.ID)
	require.NoError(t, err)

	require.Len(t, c, 1)
	require.Equal(t, res.User.ID, c[0].UserID)
	require.Equal(t, cred.PublicKey, c[0].PublicKey)
}

func TestCreateUserWithCredentials_RollbackOnCredentialFailure(t *testing.T) {
	ctx := context.Background()

	// First, create ANY user and seed a credential with a fixed ID
	existingUser := createRandomUser(t)
	fixedID := util.RandomByteArray(32)

	_ = createCredentialForUserWithID(t, existingUser.ID, fixedID)

	// Now we try to create a NEW user with a credential that has the SAME ID
	// This should cause a duplicate key violation in webauthn_credentials
	newUserHandle := util.RandomByteArray(32)
	newUserParams := NewCreateUserParams(
		"newuser_"+util.RandomString(5),
		"unique_email_"+util.RandomString(5)+"@example.com",
		nil,
		newUserHandle,
	)

	credParams := CreateCredentialsTxParams{
		ID:                      fixedID, // conflict here
		PublicKey:               util.RandomByteArray(32),
		AttestationType:         pgtype.Text{String: "none", Valid: true},
		Transports:              []byte("[]"),
		UserPresent:             true,
		UserVerified:            true,
		BackupEligible:          true,
		BackupState:             false,
		Aaguid:                  uuid.New(),
		CloneWarning:            false,
		AuthenticatorAttachment: AuthenticatorAttachmentPlatform,
		AuthenticatorData:       []byte{},
		PublicKeyAlgorithm:      -7,
	}

	_, err := testStore.CreateUserWithCredentialsTx(ctx, CreateUserWithCredentialsTxParams{
		User: newUserParams,
		Cred: credParams,
	})

	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opCreateUserWithCredentials, opErr.Op)
	require.Equal(t, entWauthnCred, opErr.Entity)
	require.Equal(t, KindConflict, opErr.Kind)

	// IMPORTANT: transaction must roll back the user insert as well.
	// Email we used above must NOT exist in DB.
	_, err = testStore.getUserByEmail(ctx, newUserParams.Email)
	require.Error(t, err) // no user created
}

func TestCreateUserWithCredentials_UserConflict(t *testing.T) {
	ctx := context.Background()

	existing := createRandomUser(t)

	u := NewCreateUserParams(existing.Username, existing.Email, nil, []byte("handle"))
	cred := credentialParamsTx()

	_, err := testStore.CreateUserWithCredentialsTx(ctx,
		CreateUserWithCredentialsTxParams{User: u, Cred: cred})

	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, entUser, opErr.Entity)
	require.Equal(t, KindConflict, opErr.Kind)

	// credential should NOT be created
	_, err = testStore.getCredentialsByID(ctx, cred.ID)
	require.Error(t, err)
}
