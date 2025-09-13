package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type PendingAuthentication struct {
	UserID      int64                 `json:"user_id"`
	Username    string                `json:"username"`
	SessionData *webauthn.SessionData `json:"session_data"`
	ExpiresAt   time.Time             `json:"expires_at"`
}

type SigninStartRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
}

type SigninStartResponse struct {
	*protocol.CredentialAssertion `json:",inline"`
}

// Function to start the sign-in process with Webauthn.
func (service *Service) signinStart(ctx *gin.Context) {
	var req SigninStartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// 1) Get user from the database, reject if not found
	user, err := service.store.GetUserByUsername(ctx, req.Username)
	if err != nil {
		// Don't reveal if user exists or not
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid credentials")))
		return
	}

	// 2) Get users creds
	creds, err := service.store.GetUserCredentials(ctx, user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 3) Creating user with creds struct to begin the auth process
	userWithCreds, err := NewUserWithCredentials(user, creds)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 4) Begin authentication
	assertion, session, err := service.webauthnConfig.BeginLogin(userWithCreds)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 5) Saving session in the Redis
	pendingAuth := PendingAuthentication{
		UserID:      user.ID,
		Username:    req.Username,
		SessionData: session,
		ExpiresAt:   time.Now().Add(service.config.AuthenticationSessionTTL),
	}

	err = service.redisStore.SaveUserAuthSession(
		ctx,
		session.Challenge,
		pendingAuth,
		service.config.AuthenticationSessionTTL,
	)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 6) Returning credential assertion to the user
	ctx.JSON(http.StatusOK, SigninStartResponse{
		CredentialAssertion: assertion,
	})
}
