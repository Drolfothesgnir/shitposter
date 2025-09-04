package api

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type User struct {
	ID              int64     `json:"id"`
	Username        string    `json:"user_name"`
	Email           string    `json:"email"`
	ProfileImageURL string    `json:"profile_img_url"`
	CreatedAt       time.Time `json:"created_at"`
	// WebauthnUserHandle []byte    `json:"-"`
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

// Data stored in memory during registration
type PendingRegistration struct {
	Email              string                `json:"email"`
	Username           string                `json:"username"`
	WebauthnUserHandle []byte                `json:"webauthn_user_handle"`
	SessionData        *webauthn.SessionData `json:"session_data"`
	ExpiresAt          time.Time             `json:"expires_at"`
}

type SignupStartRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=50"`
}

type SignupStartResponse struct {
	*protocol.CredentialCreation `json:",inline"`
}

func (service *Service) SignupStart(ctx *gin.Context) {
	var req SignupStartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// check if user with this username or email exist
	usernameExists, err := service.store.UsernameExists(ctx, req.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if usernameExists {
		err := fmt.Errorf("user with username \"%s\" already exists", req.Username)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	emailExists, err := service.store.EmailExists(ctx, req.Email)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if emailExists {
		err := fmt.Errorf("user with email \"%s\" already exists", req.Email)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// creating temporary user for webauthn
	userHandle := make([]byte, 32)
	_, err = rand.Read(userHandle)
	if err != nil {
		err := fmt.Errorf("failed to generate handle")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	tempUser := &TempUser{
		ID:                 userHandle,
		Email:              req.Email,
		Username:           req.Username,
		WebauthnUserHandle: userHandle,
	}

	create, session, err := service.webauthnConfig.BeginRegistration(tempUser)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Store registration session in Redis
	registrationData := PendingRegistration{
		Email:              req.Email,
		Username:           req.Username,
		WebauthnUserHandle: userHandle,
		SessionData:        session,
		ExpiresAt:          time.Now().Add(service.config.RegistrationSessionTTL),
	}

	err = service.redisStore.SaveUserRegSession(
		ctx,
		session.Challenge,
		registrationData,
		service.config.RegistrationSessionTTL,
	)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, SignupStartResponse{
		CredentialCreation: create,
	})
}
