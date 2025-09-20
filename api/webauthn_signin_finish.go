package api

import (
	"fmt"
	"net/http"
	"time"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/gin-gonic/gin"
)

// Paskey sign-in outline:
// 1. Client sends his username/email to the server.
// 2. Servers finds his credentials and sends challenge + metadata back to him, returns 404 otherwise.
// 3. Client signs the challenge with his private key, created during registration, and sends Webauthn response to the server.
// 4. Server checks if the challenge is solved correctly and checks sign count, then authenticates the user, returns 401 otherwise.

func (service *Service) signinFinish(ctx *gin.Context) {
	chal := ctx.GetHeader(WebauthnChallengeHeader)
	if chal == "" {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("missing challenge header")))
		return
	}

	// 1. Get pending authentication session
	pending, err := service.redisStore.GetUserAuthSession(ctx, chal)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if time.Now().After(pending.ExpiresAt) {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("registration session expired")))
		return
	}

	// 2. Load user and credentials
	user, err := service.store.GetUser(ctx, pending.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	credentials, err := service.store.GetUserCredentials(ctx, user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	userWithCreds, err := NewUserWithCredentials(user, credentials)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 3. Finish authentication
	credential, err := service.webauthnConfig.FinishLogin(userWithCreds, *pending.SessionData, ctx.Request)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(fmt.Errorf("authentication failed: %w", err)))
		return
	}

	// 4. Update credential sign count
	err = service.store.UpdateCredentialSignCount(ctx, db.UpdateCredentialSignCountParams{
		ID:        credential.ID,
		SignCount: int64(credential.Authenticator.SignCount),
	})
	if err != nil {
		// Log error but don't fail authentication
		fmt.Printf("Failed to update sign count: %v\n", err)
	}

	// 5. Generate access token
	res, err := service.generateAuthTokens(
		ctx,
		userWithCreds.User,
		ctx.Request.UserAgent(),
		ctx.ClientIP(),
	)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 6. Clean up session
	_ = service.redisStore.DeleteUserAuthSession(ctx, chal)

	// 7. Return tokens and user data to the client
	ctx.JSON(http.StatusOK, res)
}
