package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// Passkey registration process outline:
// 1. Client sends his data (username, email) to the server
// 2. Server check data validity, saves it temporary and creates "challenge" for the client to solve
// 3. Client solves the challenge, creates public and private keys, saves private for themselves and sends public to the server
// 4. Server saves user data and credentials in the db and returns user object to the client

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
func (service *Service) SignupStart(ctx *gin.Context) {
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

// Helper function to extract credential transport info from user creds or from the HTTP header.
//
// TODO: add proper data sanitizing
func extractTransportData(ctx *gin.Context, cred *webauthn.Credential) []string {
	if len(cred.Transport) > 0 {
		// need to convert native webauthn transport type to the string
		// to be consistent in return value
		transport := make([]string, len(cred.Transport))
		for i, tr := range cred.Transport {
			transport[i] = string(tr)
		}
		return transport
	}

	transport := ctx.GetHeader(WebauthnTransportHeader)
	return strings.Split(transport, ",")

}

// We only need response struct because incoming request must be raw and unparsed
// to be used in Webauthn.FinishRegistration function.
type SignupFinishResponse struct {
	User User `json:"user"`
}

func (service *Service) SignupFinish(ctx *gin.Context) {
	// I decided to get challenge as HTTP header because it is the easiest way to get it so far
	chal := ctx.GetHeader(WebauthnChallengeHeader)
	if chal == "" {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("missing challenge header")))
		return
	}

	// 1) Load pending registration session from Redis
	pending, err := service.redisStore.GetUserRegSession(ctx, chal)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if time.Now().After(pending.ExpiresAt) {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("registration session expired")))
		return
	}

	// 2) Recreate the temporary user used for BeginRegistration
	tmp := &TempUser{
		ID:                 pending.WebauthnUserHandle,
		Email:              pending.Email,
		Username:           pending.Username,
		WebauthnUserHandle: pending.WebauthnUserHandle,
	}

	// 3) Finish registration (validates challenge/origin, builds credential)
	cred, err := service.webauthnConfig.FinishRegistration(tmp, *pending.SessionData, ctx.Request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("webauthn finish failed: %w", err)))
		return
	}

	// 4) Save user data and credentials into the database

	tr := extractTransportData(ctx, cred)

	jsonTransport, err := json.Marshal(tr)
	if err != nil {
		err := fmt.Errorf("failed to marshal creds transport: %w", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	txArg := db.CreateUserWithCredentialsTxParams{
		User: db.CreateUserParams{
			Username:           pending.Username,
			Email:              pending.Email,
			WebauthnUserHandle: pending.WebauthnUserHandle,
		},
		Cred: db.CreateCredentialsTxParams{
			ID:         cred.ID,
			PublicKey:  cred.PublicKey,
			SignCount:  int64(cred.Authenticator.SignCount),
			Transports: jsonTransport,
		},
	}

	// TODO: add 23505 - unique violation check for racing transactions... but why there should be such thing
	// if it is a registration step?
	user, err := service.store.CreateUserWithCredentialsTx(ctx, txArg)
	if err != nil {
		err := fmt.Errorf("failed to save user data to the database: %w", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 5) remove tmp session ignoring possible error
	_ = service.redisStore.DeleteUserRegSession(ctx, chal)

	// 6) return user data to the client
	ctx.JSON(http.StatusOK, SignupFinishResponse{
		User: createUserResponse(user.User),
	})
}
