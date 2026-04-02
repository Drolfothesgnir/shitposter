package api

import (
	"fmt"
	"net/http"
	"time"
)

type RenewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (r RenewAccessTokenRequest) Validate() *Vomit {
	issues := make([]Issue, 0)
	validate(&issues, r.RefreshToken, "refresh_token", strRequired)
	return barf(issues)
}

type RenewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

func (server *Service) renewAccessToken(w http.ResponseWriter, r *http.Request) {
	var req RenewAccessTokenRequest
	if vErr := ingestJSONBody(w, r, &req); vErr != nil {
		respondWithJSON(w, vErr.Status, vErr)
		return
	}

	// validating the body
	if vErr := req.Validate(); vErr != nil {
		respondWithJSON(w, vErr.Status, vErr)
		return
	}

	// JWT Verification Failure
	refreshPayload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		authErr := newAuthError(
			AuthRefreshTokenErr,
			http.StatusUnauthorized,            // 401
			"invalid or expired refresh token", // Generic external message
			err,                                // Keep the raw error internal!
		)
		respondWithJSON(w, authErr.StatusCode(), authErr)
		return
	}

	ctx := r.Context()

	session, err := server.store.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		opErr := newResourceError(err)
		respondWithJSON(w, opErr.StatusCode(), opErr)
		return
	}

	if session.IsBlocked {
		// 403 IS ACTUALLY CORRECT HERE!
		// We know who they are, their token is valid, but their account is banned.
		// A 403 lets the frontend show a dedicated "Account Suspended" screen.
		authErr := newAuthError(
			AuthSessionBlocked,
			http.StatusForbidden,
			"your account has been blocked",
			nil,
		)
		respondWithJSON(w, authErr.StatusCode(), authErr)
		return
	}

	// 3. The Security Anomalies (Mismatch / Bad User)
	if session.UserID != refreshPayload.UserID || req.RefreshToken != session.RefreshToken {
		// Log this loudly internally! But tell the client exactly what we tell them
		// when a token naturally expires to give attackers zero clues.
		authErr := newAuthError(
			AuthRefreshTokenErr,
			http.StatusUnauthorized,            // 401
			"invalid or expired refresh token", // Generic external message
			fmt.Errorf("SECURITY ANOMALY: mismatch for session %s", session.ID),
		)
		respondWithJSON(w, authErr.StatusCode(), authErr)
		return
	}

	// Session Expired in DB
	if time.Now().After(session.ExpiresAt) {
		authErr := newAuthError(
			AuthSessionExpired,
			http.StatusUnauthorized,            // 401
			"invalid or expired refresh token", // Generic external message
			nil,
		)
		respondWithJSON(w, authErr.StatusCode(), authErr)
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(refreshPayload.UserID, server.config.AccessTokenDuration)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, internalResourceError())
		return
	}

	res := RenewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	}

	respondWithJSON(w, http.StatusOK, res)
}
