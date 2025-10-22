package api

import (
	"encoding/json"
	"fmt"
	"time"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type UserResponse struct {
	ID              int64     `json:"id"`
	Username        string    `json:"user_name"`
	DisplayName     string    `json:"display_name"`
	IsDeleted       bool      `json:"is_deleted"`
	DeletedAt       time.Time `json:"deleted_at"`
	Email           string    `json:"email"`
	ProfileImageURL *string   `json:"profile_img_url"`
	CreatedAt       time.Time `json:"created_at"`
}

// Helper function to map database User struct into an API response
func createUserResponse(user db.User) UserResponse {

	var profileImgUrl *string

	if user.ProfileImgUrl.Valid {
		profileImgUrl = &user.ProfileImgUrl.String
	}

	return UserResponse{
		ID:              user.ID,
		Username:        user.Username,
		DisplayName:     user.DisplayName,
		Email:           user.Email,
		ProfileImageURL: profileImgUrl,
		CreatedAt:       user.CreatedAt,
		IsDeleted:       user.IsDeleted,
		DeletedAt:       user.DeletedAt,
	}
}

// Aggregated type which implements webauthn.User interface
type UserWithCredentials struct {
	db.User
	Credentials []webauthn.Credential
}

// UserWithCredentials factory which takes raw db.User and db.WebauthnCredentials
func NewUserWithCredentials(user db.User, creds []db.WebauthnCredential) (*UserWithCredentials, error) {
	parsedCreds := make([]webauthn.Credential, len(creds))

	for i, cred := range creds {
		parsedTransport := []protocol.AuthenticatorTransport{}
		if err := json.Unmarshal(cred.Transports, &parsedTransport); err != nil {
			return nil, fmt.Errorf("failed to parse transports for credential %x: %w", cred.ID, err)
		}

		parsedCred := webauthn.Credential{
			ID:        cred.ID,
			PublicKey: cred.PublicKey,
			Transport: parsedTransport,
			Authenticator: webauthn.Authenticator{
				AAGUID:       cred.Aaguid[:],
				SignCount:    uint32(cred.SignCount),
				CloneWarning: cred.CloneWarning,
				Attachment:   protocol.AuthenticatorAttachment(cred.AuthenticatorAttachment),
			},
			AttestationType: cred.AttestationType.String,
			Attestation: webauthn.CredentialAttestation{
				AuthenticatorData:  cred.AuthenticatorData,
				PublicKeyAlgorithm: int64(cred.PublicKeyAlgorithm),
			},
			Flags: webauthn.CredentialFlags{
				UserPresent:    cred.UserPresent,
				UserVerified:   cred.UserVerified,
				BackupState:    cred.BackupState,
				BackupEligible: cred.BackupEligible,
			},
		}

		parsedCreds[i] = parsedCred
	}

	result := &UserWithCredentials{
		user,
		parsedCreds,
	}

	return result, nil
}

// following methods are required by webauthn.User interface

func (user *UserWithCredentials) WebAuthnID() []byte {
	return user.WebauthnUserHandle
}

func (user *UserWithCredentials) WebAuthnName() string {
	return user.Email
}

func (user *UserWithCredentials) WebAuthnDisplayName() string {
	return user.Username
}

func (user *UserWithCredentials) WebAuthnCredentials() []webauthn.Credential {
	return user.Credentials
}

// Temporary user for WebAuthn (not stored in DB yet)
type TempUser struct {
	ID                 []byte
	Email              string
	Username           string
	WebauthnUserHandle []byte
}

// Implement webauthn.User interface
func (u *TempUser) WebAuthnID() []byte                         { return u.WebauthnUserHandle }
func (u *TempUser) WebAuthnName() string                       { return u.Email }
func (u *TempUser) WebAuthnDisplayName() string                { return u.Username }
func (u *TempUser) WebAuthnCredentials() []webauthn.Credential { return []webauthn.Credential{} } // Empty for new user
