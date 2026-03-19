package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

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

func (service *Service) signupFinish(ctx *gin.Context) {
	// 1) Read session ID from cookie
	sessionID, err := getWebauthnSessionCookie(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, newPayloadError("missing or invalid session cookie", nil))
		return
	}

	// 2) Load pending registration session from Redis
	pending, err := service.redisStore.GetUserRegSession(ctx, sessionID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, newPayloadError("registration session not found or expired", nil))
		return
	}
	if time.Now().After(pending.ExpiresAt) {
		ctx.JSON(http.StatusBadRequest, newPayloadError("registration session expired", nil))
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
	cred, err := service.webauthnConfig.FinishRegistration(tmp, *pending.SessionData, ctx.Request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, newPayloadError("webauthn registration verification failed", nil))
		return
	}

	// 5) Save user data and credentials into the database

	tr := extractTransportData(ctx, cred)

	jsonTransport, err := json.Marshal(tr)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, internalResourceError())
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
		ctx.JSON(http.StatusInternalServerError, internalResourceError())
		return
	}

	// 6) Clean up session cookie and Redis
	secure := service.config.Environment != "development"
	clearWebauthnSessionCookie(ctx, secure)
	_ = service.redisStore.DeleteUserRegSession(ctx, sessionID)

	// 7) Return user data to the client
	res, err := service.generateAuthTokens(ctx, user, ctx.Request.UserAgent(), ctx.ClientIP())

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, internalResourceError())
		return
	}

	ctx.JSON(http.StatusOK, res)
}
