package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/Drolfothesgnir/shitposter/db/mock"
	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/token"
	mocktk "github.com/Drolfothesgnir/shitposter/token/mock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRenewAccess(t *testing.T) {

	refreshToken := "refresh_token"

	payload := &token.Payload{
		ID:        uuid.UUID{1},
		UserID:    1,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(time.Minute),
	}

	session := db.Session{
		ID:           uuid.UUID{byte(payload.UserID)},
		UserID:       payload.UserID,
		RefreshToken: refreshToken,
		UserAgent:    "Chrome",
		ClientIp:     "198.162.0.0",
		IsBlocked:    false,
		ExpiresAt:    payload.ExpiredAt,
		CreatedAt:    payload.IssuedAt,
	}

	testCases := []struct {
		name       string
		buildStubs func(
			store *mockdb.MockStore,
			tokenMaker *mocktk.MockMaker,
		)
		body          gin.H
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "MissingToken",
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktk.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(gomock.Any()).Times(0)
			},
			body: gin.H{},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidToken",
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktk.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(refreshToken).Times(1).Return(&token.Payload{}, token.ErrInvalidToken)
				store.EXPECT().GetSession(gomock.Any(), gomock.Any()).Times(0)
			},
			body: gin.H{
				"refresh_token": refreshToken,
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "TokenExpired",
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktk.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(refreshToken).Times(1).Return(&token.Payload{}, token.ErrTokenExpired)
				store.EXPECT().GetSession(gomock.Any(), gomock.Any()).Times(0)
			},
			body: gin.H{
				"refresh_token": refreshToken,
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "GetSessionNotFound",
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktk.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(refreshToken).Times(1).Return(payload, nil)
				store.EXPECT().GetSession(gomock.Any(), payload.ID).Times(1).Return(db.Session{}, pgx.ErrNoRows)
				tokenMaker.EXPECT().CreateToken(gomock.Any(), gomock.Any()).Times(0)
			},
			body: gin.H{
				"refresh_token": refreshToken,
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "GetSessionErr",
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktk.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(refreshToken).Times(1).Return(payload, nil)
				store.EXPECT().GetSession(gomock.Any(), payload.ID).Times(1).Return(db.Session{}, pgx.ErrTxClosed)
				tokenMaker.EXPECT().CreateToken(gomock.Any(), gomock.Any()).Times(0)
			},
			body: gin.H{
				"refresh_token": refreshToken,
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "SessionBlocked",
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktk.MockMaker) {
				session := db.Session{
					ID:           session.ID,
					UserID:       session.UserID,
					RefreshToken: session.RefreshToken,
					ExpiresAt:    session.ExpiresAt,
					CreatedAt:    session.CreatedAt,
					IsBlocked:    true,
				}

				tokenMaker.EXPECT().VerifyToken(refreshToken).Times(1).Return(payload, nil)
				store.EXPECT().GetSession(gomock.Any(), payload.ID).Times(1).Return(session, nil)
				tokenMaker.EXPECT().CreateToken(gomock.Any(), gomock.Any()).Times(0)
			},
			body: gin.H{
				"refresh_token": refreshToken,
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, res.Error, ErrSessionBlocked.Error())
			},
		},
		{
			name: "UserMismatch",
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktk.MockMaker) {
				session := db.Session{
					ID:           session.ID,
					UserID:       2,
					RefreshToken: session.RefreshToken,
					ExpiresAt:    session.ExpiresAt,
					CreatedAt:    session.CreatedAt,
				}

				tokenMaker.EXPECT().VerifyToken(refreshToken).Times(1).Return(payload, nil)
				store.EXPECT().GetSession(gomock.Any(), payload.ID).Times(1).Return(session, nil)
				tokenMaker.EXPECT().CreateToken(gomock.Any(), gomock.Any()).Times(0)
			},
			body: gin.H{
				"refresh_token": refreshToken,
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, res.Error, ErrSessionUserMismatch.Error())
			},
		},
		{
			name: "TokenMismatch",
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktk.MockMaker) {
				session := db.Session{
					ID:           session.ID,
					UserID:       session.UserID,
					ExpiresAt:    session.ExpiresAt,
					CreatedAt:    session.CreatedAt,
					RefreshToken: "some_other_token",
				}

				tokenMaker.EXPECT().VerifyToken(refreshToken).Times(1).Return(payload, nil)
				store.EXPECT().GetSession(gomock.Any(), payload.ID).Times(1).Return(session, nil)
				tokenMaker.EXPECT().CreateToken(gomock.Any(), gomock.Any()).Times(0)
			},
			body: gin.H{
				"refresh_token": refreshToken,
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, res.Error, ErrSessionRefreshTokenMismatch.Error())
			},
		},
		{
			name: "ExpiredSession",
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktk.MockMaker) {
				session := db.Session{
					ID:           session.ID,
					UserID:       session.UserID,
					RefreshToken: session.RefreshToken,
					CreatedAt:    session.CreatedAt,
					ExpiresAt:    time.Now().Add(-time.Minute),
				}

				tokenMaker.EXPECT().VerifyToken(refreshToken).Times(1).Return(payload, nil)
				store.EXPECT().GetSession(gomock.Any(), payload.ID).Times(1).Return(session, nil)
				tokenMaker.EXPECT().CreateToken(gomock.Any(), gomock.Any()).Times(0)
			},
			body: gin.H{
				"refresh_token": refreshToken,
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, res.Error, ErrSessionExpired.Error())
			},
		},
		{
			name: "CreateTokenErr",
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktk.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(refreshToken).Times(1).Return(payload, nil)
				store.EXPECT().GetSession(gomock.Any(), payload.ID).Times(1).Return(session, nil)
				tokenMaker.EXPECT().CreateToken(payload.UserID, time.Minute).Times(1).Return("", nil, errors.New(""))
			},
			body: gin.H{
				"refresh_token": refreshToken,
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "OK",
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktk.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(refreshToken).Times(1).Return(payload, nil)
				store.EXPECT().GetSession(gomock.Any(), payload.ID).Times(1).Return(session, nil)
				tokenMaker.EXPECT().CreateToken(payload.UserID, time.Minute).Times(1).Return("access_token", payload, nil)
			},
			body: gin.H{
				"refresh_token": refreshToken,
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// mock store
			dbCtrl := gomock.NewController(t)
			defer dbCtrl.Finish()
			store := mockdb.NewMockStore(dbCtrl)

			// mock token maker
			tkCtrl := gomock.NewController(t)
			defer tkCtrl.Finish()
			tk := mocktk.NewMockMaker(tkCtrl)

			tc.buildStubs(store, tk)

			service := newTestService(t, store, tk, nil, nil)
			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			request, err := http.NewRequest(http.MethodPost, UsersRenewAccessURL, bytes.NewReader(data))
			require.NoError(t, err)

			service.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
