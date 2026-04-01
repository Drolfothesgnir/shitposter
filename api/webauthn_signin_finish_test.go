package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/Drolfothesgnir/shitposter/db/mock"
	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/tmpstore"
	mockst "github.com/Drolfothesgnir/shitposter/tmpstore/mock"
	"github.com/Drolfothesgnir/shitposter/token"
	mocktk "github.com/Drolfothesgnir/shitposter/token/mock"
	"github.com/Drolfothesgnir/shitposter/util"
	mockwa "github.com/Drolfothesgnir/shitposter/wauthn/mock"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSigninFinish(t *testing.T) {
	sessionID := "session_id"
	username := "user1"
	userHandle := util.RandomByteArray(32)

	user := db.User{
		ID:                 1,
		Username:           username,
		WebauthnUserHandle: userHandle,
		Email:              util.RandomEmail(),
	}

	session := &webauthn.SessionData{}

	pendingAuth := &tmpstore.PendingAuthentication{
		UserID:      user.ID,
		Username:    username,
		SessionData: session,
		ExpiresAt:   time.Now().Add(time.Minute),
	}

	transports := []protocol.AuthenticatorTransport{
		protocol.USB,
		protocol.NFC,
	}

	transportsJSON, err := json.Marshal(transports)
	require.NoError(t, err)

	cred := db.WebauthnCredential{
		ID:         userHandle,
		UserID:     user.ID,
		Transports: transportsJSON,
	}

	userWithCreds, err := NewUserWithCredentials(user, []db.WebauthnCredential{cred})
	require.NoError(t, err)

	waCred := &webauthn.Credential{
		ID:        cred.ID,
		Transport: transports,
		Authenticator: webauthn.Authenticator{
			SignCount: 1,
		},
	}

	recordUseArg := db.RecordCredentialUseParams{
		ID:        waCred.ID,
		SignCount: int64(waCred.Authenticator.SignCount),
	}

	tokenStr := "token"

	tokenPayload := &token.Payload{
		ID:        uuid.New(),
		UserID:    user.ID,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(time.Minute),
	}

	sessionArg := db.CreateSessionParams{
		ID:           tokenPayload.ID,
		UserID:       tokenPayload.UserID,
		RefreshToken: tokenStr,
		UserAgent:    "chrome",
		ClientIp:     "198.162.0.0",
		IsBlocked:    false,
		ExpiresAt:    tokenPayload.ExpiredAt,
	}

	createdSession := db.Session{
		ID:           tokenPayload.ID,
		UserID:       tokenPayload.UserID,
		RefreshToken: tokenStr,
		UserAgent:    "chrome",
		ClientIp:     "198.162.0.0",
		IsBlocked:    false,
		ExpiresAt:    tokenPayload.ExpiredAt,
	}

	addCookie := func(req *http.Request) {
		req.AddCookie(&http.Cookie{
			Name:  webauthnSessionCookie,
			Value: sessionID,
		})
	}

	checkAuthFailure := func(t *testing.T, recorder *httptest.ResponseRecorder) {
		t.Helper()

		require.Equal(t, http.StatusUnauthorized, recorder.Code)

		var resp AuthError
		err := json.Unmarshal(recorder.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, KindAuth, resp.Kind)
		require.Equal(t, "authentication failed", resp.ErrMessage)
	}

	testCases := []struct {
		name       string
		buildStubs func(
			store *mockdb.MockStore,
			rs *mockst.MockStore,
			wa *mockwa.MockWebAuthnConfig,
			tokenMaker *mocktk.MockMaker,
		)
		setupRequest  func(req *http.Request)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "MissingCookie",
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig, tokenMaker *mocktk.MockMaker) {
				rs.EXPECT().GetUserAuthSession(gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest: func(req *http.Request) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "GetSessionErr",
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig, tokenMaker *mocktk.MockMaker) {
				rs.EXPECT().GetUserAuthSession(gomock.Any(), sessionID).Times(1).Return(&tmpstore.PendingAuthentication{}, errors.New(""))
				store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest: addCookie,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "SessionExpired",
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig, tokenMaker *mocktk.MockMaker) {
				expired := &tmpstore.PendingAuthentication{
					UserID:      user.ID,
					Username:    username,
					SessionData: session,
					ExpiresAt:   time.Now().Add(-time.Minute),
				}

				rs.EXPECT().GetUserAuthSession(gomock.Any(), sessionID).Times(1).Return(expired, nil)
				store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest: addCookie,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "GetUserErr",
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig, tokenMaker *mocktk.MockMaker) {
				rs.EXPECT().GetUserAuthSession(gomock.Any(), sessionID).Times(1).Return(pendingAuth, nil)
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(db.User{}, pgx.ErrNoRows)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(0)
			},
			setupRequest: addCookie,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "GetUserCredsErr",
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig, tokenMaker *mocktk.MockMaker) {
				rs.EXPECT().GetUserAuthSession(gomock.Any(), sessionID).Times(1).Return(pendingAuth, nil)
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{}, pgx.ErrNoRows)
				wa.EXPECT().FinishLogin(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest: addCookie,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "FinishLoginErr",
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig, tokenMaker *mocktk.MockMaker) {
				rs.EXPECT().GetUserAuthSession(gomock.Any(), sessionID).Times(1).Return(pendingAuth, nil)
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{cred}, nil)
				wa.EXPECT().FinishLogin(userWithCreds, *session, gomock.Any()).Times(1).Return(&webauthn.Credential{}, errors.New(""))
				store.EXPECT().RecordCredentialUse(gomock.Any(), gomock.Any()).Times(0)
				tokenMaker.EXPECT().CreateToken(gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest:  addCookie,
			checkResponse: checkAuthFailure,
		},
		{
			name: "RecordCredentialUseSecurityErr",
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig, tokenMaker *mocktk.MockMaker) {
				rs.EXPECT().GetUserAuthSession(gomock.Any(), sessionID).Times(1).Return(pendingAuth, nil)
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{cred}, nil)
				wa.EXPECT().FinishLogin(userWithCreds, *session, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().RecordCredentialUse(gomock.Any(), recordUseArg).Times(1).Return(&db.OpError{
					Kind: db.KindSecurity,
					Err:  errors.New("counter regression"),
				})
				tokenMaker.EXPECT().CreateToken(gomock.Any(), gomock.Any()).Times(0)
				rs.EXPECT().DeleteUserAuthSession(gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest:  addCookie,
			checkResponse: checkAuthFailure,
		},
		{
			name: "RecordCredentialUseNotFoundErr",
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig, tokenMaker *mocktk.MockMaker) {
				rs.EXPECT().GetUserAuthSession(gomock.Any(), sessionID).Times(1).Return(pendingAuth, nil)
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{cred}, nil)
				wa.EXPECT().FinishLogin(userWithCreds, *session, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().RecordCredentialUse(gomock.Any(), recordUseArg).Times(1).Return(&db.OpError{
					Kind: db.KindNotFound,
					Err:  errors.New("credential not found"),
				})
				tokenMaker.EXPECT().CreateToken(gomock.Any(), gomock.Any()).Times(0)
				rs.EXPECT().DeleteUserAuthSession(gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest:  addCookie,
			checkResponse: checkAuthFailure,
		},
		{
			name: "AccessTokenErr",
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig, tokenMaker *mocktk.MockMaker) {
				rs.EXPECT().GetUserAuthSession(gomock.Any(), sessionID).Times(1).Return(pendingAuth, nil)
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{cred}, nil)
				wa.EXPECT().FinishLogin(userWithCreds, *session, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().RecordCredentialUse(gomock.Any(), recordUseArg).Times(1).Return(nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("", &token.Payload{}, errors.New(""))
				tokenMaker.EXPECT().CreateToken(gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest: addCookie,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "RefreshTokenErr",
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig, tokenMaker *mocktk.MockMaker) {
				rs.EXPECT().GetUserAuthSession(gomock.Any(), sessionID).Times(1).Return(pendingAuth, nil)
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{cred}, nil)
				wa.EXPECT().FinishLogin(userWithCreds, *session, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().RecordCredentialUse(gomock.Any(), recordUseArg).Times(1).Return(nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return(tokenStr, tokenPayload, nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("", &token.Payload{}, errors.New(""))
				store.EXPECT().CreateSession(gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest: addCookie,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "CreateSessionErr",
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig, tokenMaker *mocktk.MockMaker) {
				rs.EXPECT().GetUserAuthSession(gomock.Any(), sessionID).Times(1).Return(pendingAuth, nil)
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{cred}, nil)
				wa.EXPECT().FinishLogin(userWithCreds, *session, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().RecordCredentialUse(gomock.Any(), recordUseArg).Times(1).Return(nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return(tokenStr, tokenPayload, nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return(tokenStr, tokenPayload, nil)
				store.EXPECT().CreateSession(gomock.Any(), sessionArg).Times(1).Return(db.Session{}, pgx.ErrTxClosed)
				rs.EXPECT().DeleteUserAuthSession(gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest: func(req *http.Request) {
				addCookie(req)
				req.Header.Add("User-Agent", "chrome")
				req.RemoteAddr = "198.162.0.0:12345"
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "OK",
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig, tokenMaker *mocktk.MockMaker) {
				rs.EXPECT().GetUserAuthSession(gomock.Any(), sessionID).Times(1).Return(pendingAuth, nil)
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{cred}, nil)
				wa.EXPECT().FinishLogin(userWithCreds, *session, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().RecordCredentialUse(gomock.Any(), recordUseArg).Times(1).Return(nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return(tokenStr, tokenPayload, nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return(tokenStr, tokenPayload, nil)
				store.EXPECT().CreateSession(gomock.Any(), sessionArg).Times(1).Return(createdSession, nil)
				rs.EXPECT().DeleteUserAuthSession(gomock.Any(), sessionID).Times(1).Return(nil)
			},
			setupRequest: func(req *http.Request) {
				addCookie(req)
				req.Header.Add("User-Agent", "chrome")
				req.RemoteAddr = "198.162.0.0:12345"
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "OKWithRecordCredentialUseErr",
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig, tokenMaker *mocktk.MockMaker) {
				rs.EXPECT().GetUserAuthSession(gomock.Any(), sessionID).Times(1).Return(pendingAuth, nil)
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{cred}, nil)
				wa.EXPECT().FinishLogin(userWithCreds, *session, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().RecordCredentialUse(gomock.Any(), recordUseArg).Times(1).Return(pgx.ErrTxClosed)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return(tokenStr, tokenPayload, nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return(tokenStr, tokenPayload, nil)
				store.EXPECT().CreateSession(gomock.Any(), sessionArg).Times(1).Return(createdSession, nil)
				rs.EXPECT().DeleteUserAuthSession(gomock.Any(), sessionID).Times(1).Return(nil)
			},
			setupRequest: func(req *http.Request) {
				addCookie(req)
				req.Header.Add("User-Agent", "chrome")
				req.RemoteAddr = "198.162.0.0:12345"
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbCtrl := gomock.NewController(t)
			defer dbCtrl.Finish()
			store := mockdb.NewMockStore(dbCtrl)

			rsCtrl := gomock.NewController(t)
			defer rsCtrl.Finish()
			rs := mockst.NewMockStore(rsCtrl)

			waCtrl := gomock.NewController(t)
			defer waCtrl.Finish()
			wa := mockwa.NewMockWebAuthnConfig(waCtrl)

			tkCtrl := gomock.NewController(t)
			defer tkCtrl.Finish()
			tk := mocktk.NewMockMaker(tkCtrl)

			tc.buildStubs(store, rs, wa, tk)

			service := newTestService(t, store, tk, rs, wa)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(http.MethodPost, "/users/signin/finish", nil)
			require.NoError(t, err)

			tc.setupRequest(request)

			service.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
