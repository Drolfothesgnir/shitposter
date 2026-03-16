package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type RenewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RenewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

func (server *Service) renewAccessToken(ctx *gin.Context) {
	var req RenewAccessTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, newPayloadError("invalid request parameters", err))
		return
	}

	// TODO: create proper token error mapping
	refreshPayload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, NewErrorResponse(err))
		return
	}

	session, err := server.store.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		opErr := newResourceError(err)
		ctx.JSON(opErr.StatusCode(), opErr)
		return
	}

	// TODO: create proper session error
	if session.IsBlocked {
		ctx.JSON(http.StatusUnauthorized, NewErrorResponse(ErrSessionBlocked))
		return
	}

	if session.UserID != refreshPayload.UserID {
		ctx.JSON(http.StatusUnauthorized, NewErrorResponse(ErrSessionUserMismatch))
		return
	}

	if req.RefreshToken != session.RefreshToken {
		ctx.JSON(http.StatusUnauthorized, NewErrorResponse(ErrSessionRefreshTokenMismatch))
		return
	}

	if time.Now().After(session.ExpiresAt) {
		ctx.JSON(http.StatusUnauthorized, NewErrorResponse(ErrSessionExpired))
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(refreshPayload.UserID, server.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, NewErrorResponse(err))
		return
	}

	res := RenewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	}

	ctx.JSON(http.StatusOK, res)
}
