package api

import (
	"context"
	"net/http"
	"time"

	"github.com/Drolfothesgnir/shitposter/tmpstore"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

type SigninStartRequest struct {
	Username string `json:"username"`
}

func (r SigninStartRequest) Validate() *Vomit {
	issues := make([]Issue, 0, 2)

	validate(&issues, r.Username, "username", strRequired, strMin(3), strMax(50), strAlphanum)

	return barf(issues)
}

type SigninStartResponse struct {
	*protocol.CredentialAssertion `json:",inline"`
}

// Function to start the sign-in process with Webauthn.
func (service *Service) signinStart(w http.ResponseWriter, r *http.Request) {
	var req SigninStartRequest
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

	// 1) Get user from the database, reject if not found
	user, err := service.store.GetUserByUsername(ctx, req.Username)
	if err != nil {
		opErr := newResourceError(err)
		respondWithJSON(w, opErr.StatusCode(), opErr)
		return
	}

	// 2) Get users creds
	creds, err := service.store.GetUserCredentials(ctx, user.ID)
	if err != nil {
		opErr := newResourceError(err)
		respondWithJSON(w, opErr.StatusCode(), opErr)
		return
	}

	// 3) Creating user with creds struct to begin the auth process
	userWithCreds, err := NewUserWithCredentials(user, creds)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, internalResourceError())
		return
	}

	// 4) Begin authentication
	assertion, session, err := service.webauthnConfig.BeginLogin(userWithCreds)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, internalResourceError())
		return
	}

	// 5) Saving session in Redis
	sessionID := uuid.NewString()
	pendingAuth := tmpstore.PendingAuthentication{
		UserID:      user.ID,
		Username:    req.Username,
		SessionData: session,
		ExpiresAt:   time.Now().Add(service.config.AuthenticationSessionTTL),
	}

	err = service.redisStore.SaveUserAuthSession(
		ctx,
		sessionID,
		pendingAuth,
		service.config.AuthenticationSessionTTL,
	)

	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, internalResourceError())
		return
	}

	// 6) Set session cookie and return credential assertion to the client
	service.setWebauthnSessionCookie(w, sessionID, int(service.config.AuthenticationSessionTTL.Seconds()))
	respondWithJSON(w, http.StatusOK, SigninStartResponse{
		CredentialAssertion: assertion,
	})
}
