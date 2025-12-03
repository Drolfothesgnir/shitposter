package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const opCreateUserWithCredentials = "create-user-with-credentials"

type CreateCredentialsTxParams struct {
	ID                      []byte                  `json:"id"`
	PublicKey               []byte                  `json:"public_key"`
	AttestationType         pgtype.Text             `json:"attestation_type"`
	Transports              []byte                  `json:"transports"`
	UserPresent             bool                    `json:"user_present"`
	UserVerified            bool                    `json:"user_verified"`
	BackupEligible          bool                    `json:"backup_eligible"`
	BackupState             bool                    `json:"backup_state"`
	Aaguid                  uuid.UUID               `json:"aaguid"`
	CloneWarning            bool                    `json:"clone_warning"`
	AuthenticatorAttachment AuthenticatorAttachment `json:"authenticator_attachment"`
	AuthenticatorData       []byte                  `json:"authenticator_data"`
	PublicKeyAlgorithm      int32                   `json:"public_key_algorithm"`
}

// CreateUserWithCredentialsTx creates a user and a WebAuthn credential
// in a single transaction. On error, neither the user nor the credential
// is persisted.
func NewCreateUserParams(username, email string, profileImgURL *string, webauthnUserHandle []byte) createUserParams {
	imgURL, valid := "", false
	if profileImgURL != nil {
		imgURL = *profileImgURL
		valid = true
	}

	return createUserParams{
		Username:           username,
		Email:              email,
		ProfileImgUrl:      pgtype.Text{String: imgURL, Valid: valid},
		WebauthnUserHandle: webauthnUserHandle,
	}
}

type CreateUserWithCredentialsTxParams struct {
	User createUserParams          `json:"user"`
	Cred CreateCredentialsTxParams `json:"cred"`
}

type CreateUserWithCredentialsTxResult struct {
	User User `json:"user"`
}

// Function to create both "users" row and "webauthn_credentials" row in one transaction.
func (store *SQLStore) CreateUserWithCredentialsTx(ctx context.Context, arg CreateUserWithCredentialsTxParams) (CreateUserWithCredentialsTxResult, error) {
	var result CreateUserWithCredentialsTxResult
	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		result.User, err = q.createUser(ctx, arg.User)
		if err != nil {
			return sqlError(
				opCreateUserWithCredentials,
				opDetails{
					entity: entUser,
				},
				err,
			)
		}

		params := createWebauthnCredentialsParams{
			ID:                      arg.Cred.ID,
			UserID:                  result.User.ID,
			PublicKey:               arg.Cred.PublicKey,
			SignCount:               0,
			Transports:              arg.Cred.Transports,
			AttestationType:         arg.Cred.AttestationType,
			UserPresent:             arg.Cred.UserPresent,
			UserVerified:            arg.Cred.UserVerified,
			BackupEligible:          arg.Cred.BackupEligible,
			BackupState:             arg.Cred.BackupState,
			Aaguid:                  arg.Cred.Aaguid,
			CloneWarning:            arg.Cred.CloneWarning,
			AuthenticatorAttachment: arg.Cred.AuthenticatorAttachment,
			AuthenticatorData:       arg.Cred.AuthenticatorData,
			PublicKeyAlgorithm:      arg.Cred.PublicKeyAlgorithm,
		}

		_, err = q.createWebauthnCredentials(ctx, params)

		if err != nil {
			return sqlError(
				opCreateUserWithCredentials,
				opDetails{entity: entWauthnCred, userID: result.User.ID},
				err,
			)
		}
		return nil
	})

	return result, err
}
