package api

import (
	"encoding/json"
	"fmt"
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
	res, err := service.generateAuthTokens(ctx, user.User, ctx.Request.UserAgent(), ctx.ClientIP())

	if err != nil {
		err := fmt.Errorf("failed to generate auth tokens: %w", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, res)
}
