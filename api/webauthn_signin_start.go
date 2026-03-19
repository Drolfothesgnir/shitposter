package api

import (
	"net/http"
	"time"

	"github.com/Drolfothesgnir/shitposter/tmpstore"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

type SigninStartRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50,alphanum"`
}

type SigninStartResponse struct {
	*protocol.CredentialAssertion `json:",inline"`
}

// Function to start the sign-in process with Webauthn.
func (service *Service) signinStart(ctx *gin.Context) {
	var req SigninStartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, newPayloadError("invalid request parameters", err))
		return
	}

	// 1) Get user from the database, reject if not found
	user, err := service.store.GetUserByUsername(ctx, req.Username)
	if err != nil {
		opErr := newResourceError(err)
		ctx.JSON(opErr.StatusCode(), opErr)
		return
	}

	// 2) Get users creds
	creds, err := service.store.GetUserCredentials(ctx, user.ID)
	if err != nil {
		opErr := newResourceError(err)
		ctx.JSON(opErr.StatusCode(), opErr)
		return
	}

	// 3) Creating user with creds struct to begin the auth process
	userWithCreds, err := NewUserWithCredentials(user, creds)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, internalResourceError())
		return
	}

	// 4) Begin authentication
	assertion, session, err := service.webauthnConfig.BeginLogin(userWithCreds)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, internalResourceError())
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
		ctx.JSON(http.StatusInternalServerError, internalResourceError())
		return
	}

	// 6) Set session cookie and return credential assertion to the client
	service.setWebauthnSessionCookie(ctx, sessionID, int(service.config.AuthenticationSessionTTL.Seconds()))
	ctx.JSON(http.StatusOK, SigninStartResponse{
		CredentialAssertion: assertion,
	})
}
