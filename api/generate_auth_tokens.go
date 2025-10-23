package api

import (
	"context"
	"time"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/google/uuid"
)

type PrivateSuccessAuthResponse struct {
	SessionID             uuid.UUID           `json:"session_id"`
	AccessToken           string              `json:"access_token"`
	AccessTokenExpiresAt  time.Time           `json:"access_token_expires_at"`
	RefreshToken          string              `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time           `json:"refresh_token_expires_at"`
	User                  PrivateUserResponse `json:"user"`
}

func (service *Service) generateAuthTokens(ctx context.Context, user db.User, userAgent, clientIP string) (*PrivateSuccessAuthResponse, error) {
	accessToken, accessPayload, err := service.tokenMaker.CreateToken(user.ID, service.config.AccessTokenDuration)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshPayload, err := service.tokenMaker.CreateToken(user.ID, service.config.RefreshTokenDuration)
	if err != nil {
		return nil, err
	}

	sessionParams := db.CreateSessionParams{
		ID:           refreshPayload.ID,
		UserID:       user.ID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIp:     clientIP,
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	}

	session, err := service.store.CreateSession(ctx, sessionParams)

	if err != nil {
		return nil, err
	}

	res := &PrivateSuccessAuthResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  createPrivateUserResponse(user),
	}

	return res, nil
}
