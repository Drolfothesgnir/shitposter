package api

import (
	"errors"
	"net/http"
	"time"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Paskey sign-in outline:
// 1. Client sends his username/email to the server.
// 2. Servers finds his credentials and sends challenge + metadata back to him, returns 404 otherwise.
// 3. Client signs the challenge with his private key, created during registration, and sends Webauthn response to the server.
// 4. Server checks if the challenge is solved correctly and checks sign count, then authenticates the user, returns 401 otherwise.

func (service *Service) signinFinish(ctx *gin.Context) {
	// 1. Read session ID from cookie
	sessionID, err := getWebauthnSessionCookie(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, newPayloadError("missing or invalid session cookie", nil))
		return
	}

	// 2. Get pending authentication session
	pending, err := service.redisStore.GetUserAuthSession(ctx, sessionID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, newPayloadError("authentication session not found or expired", nil))
		return
	}
	if time.Now().After(pending.ExpiresAt) {
		ctx.JSON(http.StatusBadRequest, newPayloadError("authentication session expired", nil))
		return
	}

	// 3. Load user and credentials
	user, err := service.store.GetUser(ctx, pending.UserID)
	if err != nil {
		opErr := newResourceError(err)
		ctx.JSON(opErr.StatusCode(), opErr)
		return
	}

	credentials, err := service.store.GetUserCredentials(ctx, user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, internalResourceError())
		return
	}

	userWithCreds, err := NewUserWithCredentials(user, credentials)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, internalResourceError())
		return
	}

	// 4. Finish authentication
	credential, err := service.webauthnConfig.FinishLogin(userWithCreds, *pending.SessionData, ctx.Request)
	if err != nil {
		authErr := newAuthError(AuthInternal, "authentication failed")
		ctx.JSON(authErr.StatusCode(), authErr)
		return
	}

	// 5. Update credential sign count
	// TODO: add policy comments
	err = service.store.RecordCredentialUse(ctx, db.RecordCredentialUseParams{
		ID:        credential.ID,
		SignCount: int64(credential.Authenticator.SignCount),
	})
	if err != nil {
		var opErr *db.OpError
		if errors.As(err, &opErr) && (opErr.Kind == db.KindSecurity || opErr.Kind == db.KindNotFound) {
			log.Warn().
				Err(err).
				Str("kind", opErr.Kind.String()).
				Msg("Rejecting authentication after credential use check")

			authErr := newAuthError(AuthInternal, "authentication failed")
			ctx.JSON(authErr.StatusCode(), authErr)
			return
		}
	}

	// 6. Generate access token
	res, err := service.generateAuthTokens(
		ctx,
		userWithCreds.User,
		ctx.Request.UserAgent(),
		ctx.ClientIP(),
	)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, internalResourceError())
		return
	}

	// 7. Clean up session cookie and Redis
	secure := service.config.Environment != "development"
	clearWebauthnSessionCookie(ctx, secure)
	_ = service.redisStore.DeleteUserAuthSession(ctx, sessionID)

	// 8. Return tokens and user data to the client
	ctx.JSON(http.StatusOK, res)
}
