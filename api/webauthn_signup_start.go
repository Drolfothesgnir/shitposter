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

// Passkey registration process outline:
// 1. Client sends his data (username, email) to the server
// 2. Server check data validity, saves it temporary and creates "challenge" for the client to solve
// 3. Client solves the challenge, creates public and private keys, saves private for themselves and sends public to the server
// 4. Server saves user data and credentials in the db and returns user object to the client

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

// TODO: add profile image handling during registration
func (service *Service) signupStart(ctx *gin.Context) {
	var req SignupStartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// 1) check if provided username and email are unique, reject with 400 otherwise
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

	// 2) creating temporary user for webauthn
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

	// 3) init registration process with temporary user
	create, session, err := service.webauthnConfig.BeginRegistration(tempUser)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 4) Store registration session in Redis
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

	// 5) return challenge and options to the user
	ctx.JSON(http.StatusOK, SignupStartResponse{
		CredentialCreation: create,
	})
}
