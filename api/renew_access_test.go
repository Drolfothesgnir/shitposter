package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRenewAccess(t *testing.T) {

	refreshToken := "refresh_token"

	payload := &token.Payload{
		ID:        uuid.New(),
		UserID:    1,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(time.Minute),
	}

	session := db.Session{
		ID:           uuid.New(),
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
				var resp AuthError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindAuth, resp.Kind)
				require.Equal(t, token.ErrInvalidToken.Error(), resp.Error)
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
				var resp AuthError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindAuth, resp.Kind)
				require.Equal(t, token.ErrTokenExpired.Error(), resp.Error)
			},
		},
		{
			name: "GetSessionNotFound",
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktk.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(refreshToken).Times(1).Return(payload, nil)
				store.EXPECT().GetSession(gomock.Any(), payload.ID).Times(1).Return(
					db.Session{},
					&db.OpError{
						Op:       "get-session",
						Kind:     db.KindNotFound,
						Entity:   "session",
						EntityID: payload.ID.String(),
						Err:      fmt.Errorf("session not found"),
					},
				)
				tokenMaker.EXPECT().CreateToken(gomock.Any(), gomock.Any()).Times(0)
			},
			body: gin.H{
				"refresh_token": refreshToken,
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, "not_found", resp.Reason)
			},
		},
		{
			name: "GetSessionErr",
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktk.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(refreshToken).Times(1).Return(payload, nil)
				store.EXPECT().GetSession(gomock.Any(), payload.ID).Times(1).Return(
					db.Session{},
					&db.OpError{
						Op:     "get-session",
						Kind:   db.KindInternal,
						Entity: "session",
						Err:    fmt.Errorf("tx closed"),
					},
				)
				tokenMaker.EXPECT().CreateToken(gomock.Any(), gomock.Any()).Times(0)
			},
			body: gin.H{
				"refresh_token": refreshToken,
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, "internal", resp.Reason)
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
				var resp AuthError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindAuth, resp.Kind)
				require.Equal(t, ErrSessionBlocked.Error(), resp.Error)
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
				var resp AuthError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindAuth, resp.Kind)
				require.Equal(t, ErrSessionUserMismatch.Error(), resp.Error)
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
				var resp AuthError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindAuth, resp.Kind)
				require.Equal(t, ErrSessionRefreshTokenMismatch.Error(), resp.Error)
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
				var resp AuthError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindAuth, resp.Kind)
				require.Equal(t, ErrSessionExpired.Error(), resp.Error)
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
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, "internal", resp.Reason)
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

			request, err := http.NewRequest(http.MethodPost, "/users/renew_access", bytes.NewReader(data))
			require.NoError(t, err)

			service.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
