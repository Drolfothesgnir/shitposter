package api

import (
	"time"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
)

type User struct {
	ID              int64     `json:"id"`
	Username        string    `json:"user_name"`
	Email           string    `json:"email"`
	ProfileImageURL *string   `json:"profile_img_url"`
	CreatedAt       time.Time `json:"created_at"`
}

// Helper function to map database User struct into an API response
func createUserResponse(user db.User) User {

	var profileImgUrl *string

	if user.ProfileImgUrl.Valid {
		profileImgUrl = &user.ProfileImgUrl.String
	}

	return User{
		ID:              user.ID,
		Username:        user.Username,
		Email:           user.Email,
		ProfileImageURL: profileImgUrl,
		CreatedAt:       user.CreatedAt,
	}
}

// // Aggregated type which implements webauthn.User interface
// type UserWithCredentials struct {
// 	db.User
// 	Credentials []webauthn.Credential
// }

// // UserWithCredentials factory which takes raw db.User and db.WebauthnCredentials
// func NewUserWithCredentials(user db.User, creds []db.WebauthnCredential) (*UserWithCredentials, error) {
// 	parsedCreds := make([]webauthn.Credential, len(creds))

// 	for i, cred := range creds {
// 		parsedTransport := []protocol.AuthenticatorTransport{}
// 		if err := json.Unmarshal(cred.Transports, &parsedTransport); err != nil {
// 			return nil, fmt.Errorf("failed to parse transports for credential %x: %w", cred.ID, err)
// 		}

// 		parsedCred := webauthn.Credential{
// 			ID:        cred.ID,
// 			PublicKey: cred.PublicKey,
// 			Transport: parsedTransport,
// 			Authenticator: webauthn.Authenticator{
// 				AAGUID:    []byte{}, // Don't care about device type
// 				SignCount: uint32(cred.SignCount),
// 			},
// 			AttestationType: "none", // Don't care about device type
// 			Flags: webauthn.CredentialFlags{
// 				UserPresent:  true, // User confirmed action
// 				UserVerified: true, // User provided biometric/PIN
// 			},
// 		}

// 		parsedCreds[i] = parsedCred
// 	}

// 	result := &UserWithCredentials{
// 		user,
// 		parsedCreds,
// 	}

// 	return result, nil
// }

// // following methods are required by webauthn.User interface

// func (user *UserWithCredentials) WebAuthnID() []byte {
// 	return user.WebauthnUserHandle
// }

// func (user *UserWithCredentials) WebAuthnName() string {
// 	return user.Email
// }

// func (user *UserWithCredentials) WebAuthnDisplayName() string {
// 	return user.Username
// }

// func (user *UserWithCredentials) WebAuthnCredentials() []webauthn.Credential {
// 	return user.Credentials
// }
