package api

import (
	"errors"
	"net/http"
	"time"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/rs/zerolog/log"
)

// Paskey sign-in outline:
// 1. Client sends his username/email to the server.
// 2. Servers finds his credentials and sends challenge + metadata back to him, returns 404 otherwise.
// 3. Client signs the challenge with his private key, created during registration, and sends Webauthn response to the server.
// 4. Server checks if the challenge is solved correctly and checks sign count, then authenticates the user, returns 401 otherwise.

func (service *Service) signinFinish(w http.ResponseWriter, r *http.Request) {
	// 1. Read session ID from cookie
	sessionID := getWebauthnSessionCookieValue(r)
	if sessionID == "" {
		// We use AuthError here because this is an authentication state failure,
		// not a JSON body parsing/validation failure.
		aErr := newAuthError(
			AuthSessionNotFound,   // The flavor we added in the previous step
			http.StatusBadRequest, // 400 because they submitted a malformed request for this stage
			"missing or invalid session cookie",
			nil, // No underlying Go error to log here
		)
		abortWithError(w, aErr)
		return
	}

	ctx := r.Context()

	// 2. Get pending authentication session
	pending, err := service.redisStore.GetUserAuthSession(ctx, sessionID)
	if err != nil {
		aErr := newAuthError(
			AuthSessionNotFound,
			http.StatusBadRequest,
			"authentication session not found or expired",
			nil,
		)
		abortWithError(w, aErr)
		return
	}

	if time.Now().After(pending.ExpiresAt) {
		aErr := newAuthError(
			AuthSessionExpired,
			http.StatusUnauthorized,
			"authentication session expired",
			nil,
		)
		abortWithError(w, aErr)
		return
	}

	// 3. Load user and credentials
	user, err := service.store.GetUser(ctx, pending.UserID)
	if err != nil {
		opErr := newResourceError(err)
		respondWithJSON(w, opErr.StatusCode(), opErr)
		return
	}

	credentials, err := service.store.GetUserCredentials(ctx, user.ID)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, internalResourceError())
		return
	}

	userWithCreds, err := NewUserWithCredentials(user, credentials)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, internalResourceError())
		return
	}

	// 4. Finish authentication
	credential, err := service.webauthnConfig.FinishLogin(userWithCreds, *pending.SessionData, r)
	if err != nil {
		authErr := newAuthError(
			FlavorInternal,
			http.StatusUnauthorized,
			"authentication failed",
			err)
		respondWithJSON(w, authErr.StatusCode(), authErr)
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

			authErr := newAuthError(
				AuthVerificationFailed,
				http.StatusUnauthorized,
				"authentication failed",
				opErr)
			respondWithJSON(w, authErr.StatusCode(), authErr)
			return
		}
	}

	// 6. Generate access token
	res, err := service.generateAuthTokens(
		ctx,
		userWithCreds.User,
		r.UserAgent(),
		getClientIP(r),
	)

	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, internalResourceError())
		return
	}

	// 7. Clean up session cookie and Redis
	secure := service.config.Environment != "development"
	clearWebauthnSessionCookie(w, secure)
	_ = service.redisStore.DeleteUserAuthSession(ctx, sessionID)

	// 8. Return tokens and user data to the client
	respondWithJSON(w, http.StatusOK, res)
}
