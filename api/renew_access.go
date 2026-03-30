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

	refreshPayload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		authErr := newAuthError(err.Error())
		ctx.JSON(authErr.StatusCode(), authErr)
		return
	}

	session, err := server.store.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		opErr := newResourceError(err)
		ctx.JSON(opErr.StatusCode(), opErr)
		return
	}

	if session.IsBlocked {
		authErr := newAuthError("session is blocked")
		ctx.JSON(authErr.StatusCode(), authErr)
		return
	}

	if session.UserID != refreshPayload.UserID {
		authErr := newAuthError("incorrect session user")
		ctx.JSON(authErr.StatusCode(), authErr)
		return
	}

	if req.RefreshToken != session.RefreshToken {
		authErr := newAuthError("refresh token mismatch")
		ctx.JSON(authErr.StatusCode(), authErr)
		return
	}

	if time.Now().After(session.ExpiresAt) {
		authErr := newAuthError("session is expired")
		ctx.JSON(authErr.StatusCode(), authErr)
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(refreshPayload.UserID, server.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, internalResourceError())
		return
	}

	res := RenewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	}

	ctx.JSON(http.StatusOK, res)
}
