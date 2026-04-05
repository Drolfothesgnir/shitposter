package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Helper function to extract credential transport info from user creds or from the HTTP header.
//
// TODO: add proper data sanitizing
func extractTransportData(r *http.Request, cred *webauthn.Credential) []string {
	if len(cred.Transport) > 0 {
		// need to convert native webauthn transport type to the string
		// to be consistent in return value
		transport := make([]string, len(cred.Transport))
		for i, tr := range cred.Transport {
			transport[i] = string(tr)
		}
		return transport
	}

	transport := r.Header.Get(WebauthnTransportHeader)
	return strings.Split(transport, ",")

}

func (service *Service) signupFinish(w http.ResponseWriter, r *http.Request) {
	// 1) Read session ID from cookie
	sessionID := getWebauthnSessionCookieValue(r)
	if sessionID == "" {
		vErr := puke(ReqMissingData, http.StatusBadRequest, "missing or invalid session cookie", nil)
		respondWithJSON(w, vErr.Status, vErr)
		return
	}

	// FIX: Always use the request context to respect client disconnects/timeouts
	ctx := r.Context()

	// 2) Load pending registration session from Redis
	pending, err := service.redisStore.GetUserRegSession(ctx, sessionID)
	if err != nil {
		vErr := puke(AuthSessionNotFound, http.StatusBadRequest, "registration session not found or expired", err)
		respondWithJSON(w, vErr.Status, vErr)
		return
	}

	if time.Now().After(pending.ExpiresAt) {
		vErr := puke(AuthSessionExpired, http.StatusBadRequest, "registration session expired", nil)
		respondWithJSON(w, vErr.Status, vErr)
		return
	}

	// 3) Recreate the temporary user used for BeginRegistration
	tmp := &TempUser{
		ID:                 pending.WebauthnUserHandle,
		Email:              pending.Email,
		Username:           pending.Username,
		WebauthnUserHandle: pending.WebauthnUserHandle,
	}

	// 4) Finish registration (validates challenge/origin, builds credential)
	cred, err := service.webauthnConfig.FinishRegistration(tmp, *pending.SessionData, r)
	if err != nil {
		vErr := puke(AuthVerificationFailed, http.StatusBadRequest, "webauthn registration verification failed", err)
		respondWithJSON(w, vErr.Status, vErr)
		return
	}

	// 5) Save user data and credentials into the database
	tr := extractTransportData(r, cred)

	jsonTransport, err := json.Marshal(tr)
	if err != nil {
		// Assuming internalResourceError() returns a *Vomit or you can replace it with puke()
		respondWithJSON(w, http.StatusInternalServerError, internalResourceError())
		return
	}

	txArg := db.CreateUserWithCredentialsTxParams{
		User: db.NewCreateUserParams(
			pending.Username,
			pending.Email,
			nil, // TODO: somehow provide profile image during registration
			pending.WebauthnUserHandle,
		),
		Cred: db.CreateCredentialsTxParams{
			ID:                      cred.ID,
			PublicKey:               cred.PublicKey,
			Transports:              jsonTransport,
			AttestationType:         pgtype.Text{String: cred.AttestationType, Valid: cred.AttestationType != ""},
			UserPresent:             cred.Flags.UserPresent,
			UserVerified:            cred.Flags.UserVerified,
			BackupEligible:          cred.Flags.BackupEligible,
			BackupState:             cred.Flags.BackupState,
			Aaguid:                  uuid.UUID(cred.Authenticator.AAGUID),
			CloneWarning:            cred.Authenticator.CloneWarning,
			AuthenticatorAttachment: db.AuthenticatorAttachment(cred.Authenticator.Attachment),
			AuthenticatorData:       cred.Attestation.AuthenticatorData,
			PublicKeyAlgorithm:      int32(cred.Attestation.PublicKeyAlgorithm),
		},
	}

	// TODO: add 23505 - unique violation check for racing transactions
	user, err := service.store.CreateUserWithCredentialsTx(ctx, txArg)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, internalResourceError())
		return
	}

	// 6) Clean up session cookie and Redis
	secure := service.config.Environment != "development"
	clearWebauthnSessionCookie(w, secure)
	_ = service.redisStore.DeleteUserRegSession(ctx, sessionID)

	// 7) Return user data to the client
	res, err := service.generateAuthTokens(ctx, user, r.UserAgent(), getClientIP(r))
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, internalResourceError())
		return
	}

	respondWithJSON(w, http.StatusOK, res)
}
