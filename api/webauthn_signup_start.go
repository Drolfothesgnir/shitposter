package api

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/Drolfothesgnir/shitposter/tmpstore"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

// Passkey registration process outline:
// 1. Client sends his data (username, email) to the server
// 2. Server check data validity, saves it temporary and creates "challenge" for the client to solve
// 3. Client solves the challenge, creates public and private keys, saves private for themselves and sends public to the server
// 4. Server saves user data and credentials in the db and returns user object to the client

type SignupStartRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

func (r SignupStartRequest) Validate() *Vomit {
	// email -> required OR pattern + username -> required OR min OR max AND alphanum = 3 possible errors
	issues := make([]Issue, 0, 3)

	// email
	validate(&issues, r.Email, "email", strRequired, strEmail)

	// username
	validate(&issues, r.Username, "username", strRequired, strMin(3), strMax(50), strAlphanum)

	return barf(issues)
}

type SignupStartResponse struct {
	*protocol.CredentialCreation `json:",inline"`
}

// TODO: add profile image handling during registration
func (service *Service) signupStart(w http.ResponseWriter, r *http.Request) {
	// tasting the body
	var req SignupStartRequest
	if vErr := ingestJSONBody(w, r, &req); vErr != nil {
		respondWithJSON(w, vErr.Status, vErr)
		return
	}

	// validating the body
	if vErr := req.Validate(); vErr != nil {
		respondWithJSON(w, vErr.Status, vErr)
		return
	}

	ctx := context.Background()

	// 1) check if provided username and email are unique, reject with 400 otherwise
	usernameExists, err := service.store.UsernameExists(ctx, req.Username)
	if err != nil {
		opErr := newResourceError(err)
		respondWithJSON(w, opErr.StatusCode(), opErr)
		return
	}

	if usernameExists {
		v := puke(
			ReqInvalidArguments,
			http.StatusConflict,
			fmt.Sprintf("user with username [%s] already exists", req.Username),
			nil,
		)
		respondWithJSON(w, v.Status, v)
		return
	}

	emailExists, err := service.store.EmailExists(ctx, req.Email)
	if err != nil {
		opErr := newResourceError(err)
		respondWithJSON(w, opErr.StatusCode(), opErr)
		return
	}

	if emailExists {
		v := puke(
			ReqInvalidArguments,
			http.StatusConflict,
			fmt.Sprintf("user with email [%s] already exists", req.Email),
			nil,
		)
		respondWithJSON(w, v.Status, v)
		return
	}

	// 2) creating temporary user for webauthn
	userHandle := make([]byte, 32)
	_, err = rand.Read(userHandle)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, internalResourceError())
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
		respondWithJSON(w, http.StatusInternalServerError, internalResourceError())
		return
	}

	// 4) Store registration session in Redis
	sessionID := uuid.NewString()
	registrationData := tmpstore.PendingRegistration{
		Email:              req.Email,
		Username:           req.Username,
		WebauthnUserHandle: userHandle,
		SessionData:        session,
		ExpiresAt:          time.Now().Add(service.config.RegistrationSessionTTL),
	}

	err = service.redisStore.SaveUserRegSession(
		ctx,
		sessionID,
		registrationData,
		service.config.RegistrationSessionTTL,
	)

	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, internalResourceError())
		return
	}

	// 5) Set session cookie and return challenge options to the client
	service.setWebauthnSessionCookie(w, sessionID, int(service.config.RegistrationSessionTTL.Seconds()))
	respondWithJSON(w, http.StatusOK, SignupStartResponse{
		CredentialCreation: create,
	})
}
