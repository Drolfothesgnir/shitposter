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
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSignupFinish(t *testing.T) {

	sessionID := "session_id"

	userHandle := util.RandomByteArray(32)

	aaguid := util.RandomByteArray(16)

	transports := []protocol.AuthenticatorTransport{
		protocol.USB,
		protocol.NFC,
	}

	transportsJson, err := json.Marshal(transports)
	require.NoError(t, err)

	txArg := db.CreateUserWithCredentialsTxParams{
		User: db.NewCreateUserParams(
			util.RandomOwner(),
			util.RandomEmail(),
			nil,
			userHandle,
		),
		Cred: db.CreateCredentialsTxParams{
			ID:                      userHandle,
			PublicKey:               util.RandomByteArray(16),
			AttestationType:         pgtype.Text{String: "internal", Valid: true},
			Transports:              transportsJson,
			UserPresent:             true,
			UserVerified:            true,
			BackupEligible:          true,
			BackupState:             true,
			Aaguid:                  uuid.UUID(aaguid),
			CloneWarning:            false,
			AuthenticatorAttachment: db.AuthenticatorAttachment(protocol.Platform),
			AuthenticatorData:       []byte{},
			PublicKeyAlgorithm:      -7,
		},
	}

	tmpUser := &TempUser{
		ID:                 userHandle,
		Email:              txArg.User.Email,
		Username:           txArg.User.Username,
		WebauthnUserHandle: txArg.User.WebauthnUserHandle,
	}

	waCred := &webauthn.Credential{
		ID:              userHandle,
		PublicKey:       txArg.Cred.PublicKey,
		AttestationType: txArg.Cred.AttestationType.String,
		Transport:       transports,
		Flags:           webauthn.NewCredentialFlags(255),
		Authenticator: webauthn.Authenticator{
			AAGUID:       aaguid,
			SignCount:    0,
			CloneWarning: false,
			Attachment:   protocol.Platform,
		},
		Attestation: webauthn.CredentialAttestation{
			PublicKeyAlgorithm: -7,
			AuthenticatorData:  []byte{},
		},
	}

	pending := &tmpstore.PendingRegistration{
		ExpiresAt:          time.Now().Add(time.Hour),
		SessionData:        &webauthn.SessionData{},
		Email:              txArg.User.Email,
		Username:           txArg.User.Username,
		WebauthnUserHandle: userHandle,
	}

	user := db.User{
		ID:                 1,
		Username:           txArg.User.Username,
		WebauthnUserHandle: userHandle,
		Email:              txArg.User.Email,
	}

	tokenPayload := &token.Payload{
		ID:        uuid.New(),
		UserID:    user.ID,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(time.Minute),
	}

	sessionArg := db.CreateSessionParams{
		ID:           tokenPayload.ID,
		UserID:       tokenPayload.UserID,
		RefreshToken: "refresh_token",
		UserAgent:    "chrome",
		ClientIp:     "198.162.0.0",
		IsBlocked:    false,
		ExpiresAt:    tokenPayload.ExpiredAt,
	}

	createdSession := db.Session{
		ID:           tokenPayload.ID,
		UserID:       tokenPayload.UserID,
		RefreshToken: "refresh_token",
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

	checkVomit := func(t *testing.T, recorder *httptest.ResponseRecorder, expected Vomit) {
		t.Helper()

		require.Equal(t, expected.Status, recorder.Code)

		var resp Vomit
		err := json.NewDecoder(recorder.Body).Decode(&resp)
		require.NoError(t, err)
		require.Equal(t, expected, resp)
	}

	checkInternalResourceError := func(t *testing.T, recorder *httptest.ResponseRecorder) {
		t.Helper()

		require.Equal(t, http.StatusInternalServerError, recorder.Code)

		var resp ResourceError
		err := json.NewDecoder(recorder.Body).Decode(&resp)
		require.NoError(t, err)
		require.Equal(t, ResourceError{
			Kind:   KindResource,
			Reason: db.KindInternal.String(),
			Status: http.StatusInternalServerError,
			Error:  "an internal error occurred",
		}, resp)
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
			buildStubs: func(
				store *mockdb.MockStore,
				rs *mockst.MockStore,
				wa *mockwa.MockWebAuthnConfig,
				tokenMaker *mocktk.MockMaker,
			) {
				rs.EXPECT().GetUserRegSession(gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest: func(req *http.Request) {
				// no cookie — test the missing cookie path
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkVomit(t, recorder, Vomit{
					Kind:       KindPayload,
					Reason:     ReqMissingData,
					Status:     http.StatusBadRequest,
					ErrMessage: "missing or invalid session cookie",
				})
			},
		},
		{
			name: "GetRegSessionErr",
			buildStubs: func(
				store *mockdb.MockStore,
				rs *mockst.MockStore,
				wa *mockwa.MockWebAuthnConfig,
				tokenMaker *mocktk.MockMaker,
			) {
				session := &tmpstore.PendingRegistration{}
				rs.EXPECT().GetUserRegSession(gomock.Any(), sessionID).Times(1).Return(session, errors.New(""))
				wa.EXPECT().FinishRegistration(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest: addCookie,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkVomit(t, recorder, Vomit{
					Kind:       KindPayload,
					Reason:     AuthSessionNotFound,
					Status:     http.StatusBadRequest,
					ErrMessage: "registration session not found or expired",
				})
			},
		},
		{
			name: "SessionExpired",
			buildStubs: func(
				store *mockdb.MockStore,
				rs *mockst.MockStore,
				wa *mockwa.MockWebAuthnConfig,
				tokenMaker *mocktk.MockMaker,
			) {
				session := &tmpstore.PendingRegistration{
					ExpiresAt: time.Now().Add(-time.Hour),
				}
				rs.EXPECT().GetUserRegSession(gomock.Any(), sessionID).Times(1).Return(session, nil)
				wa.EXPECT().FinishRegistration(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest: addCookie,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkVomit(t, recorder, Vomit{
					Kind:       KindPayload,
					Reason:     AuthSessionExpired,
					Status:     http.StatusBadRequest,
					ErrMessage: "registration session expired",
				})
			},
		},
		{
			name: "FinishregistrationErr",
			buildStubs: func(
				store *mockdb.MockStore,
				rs *mockst.MockStore,
				wa *mockwa.MockWebAuthnConfig,
				tokenMaker *mocktk.MockMaker,
			) {
				rs.EXPECT().GetUserRegSession(gomock.Any(), sessionID).Times(1).Return(pending, nil)
				wa.EXPECT().FinishRegistration(tmpUser, *pending.SessionData, gomock.Any()).Times(1).Return(&webauthn.Credential{}, errors.New(""))
				store.EXPECT().CreateUserWithCredentialsTx(gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest: addCookie,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkVomit(t, recorder, Vomit{
					Kind:       KindPayload,
					Reason:     AuthVerificationFailed,
					Status:     http.StatusBadRequest,
					ErrMessage: "webauthn registration verification failed",
				})
			},
		},
		{
			name: "CreateCredsErr",
			buildStubs: func(
				store *mockdb.MockStore,
				rs *mockst.MockStore,
				wa *mockwa.MockWebAuthnConfig,
				tokenMaker *mocktk.MockMaker,
			) {
				rs.EXPECT().GetUserRegSession(gomock.Any(), sessionID).Times(1).Return(pending, nil)
				wa.EXPECT().FinishRegistration(tmpUser, *pending.SessionData, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().CreateUserWithCredentialsTx(gomock.Any(), txArg).Times(1).Return(db.User{}, pgx.ErrTxClosed)
				rs.EXPECT().DeleteUserRegSession(gomock.Any(), sessionID).Times(0)
			},
			setupRequest: addCookie,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkInternalResourceError(t, recorder)
			},
		},
		{
			name: "CreateAccessTokenErr",
			buildStubs: func(
				store *mockdb.MockStore,
				rs *mockst.MockStore,
				wa *mockwa.MockWebAuthnConfig,
				tokenMaker *mocktk.MockMaker,
			) {
				rs.EXPECT().GetUserRegSession(gomock.Any(), sessionID).Times(1).Return(pending, nil)
				wa.EXPECT().FinishRegistration(tmpUser, *pending.SessionData, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().CreateUserWithCredentialsTx(gomock.Any(), txArg).Times(1).Return(user, nil)
				rs.EXPECT().DeleteUserRegSession(gomock.Any(), sessionID).Times(1).Return(nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("", &token.Payload{}, errors.New(""))
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(0)
			},
			setupRequest: addCookie,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkInternalResourceError(t, recorder)
				require.Contains(t, recorder.Header().Get("Set-Cookie"), webauthnSessionCookie+"=;")
			},
		},
		{
			name: "CreateRefreshTokenErr",
			buildStubs: func(
				store *mockdb.MockStore,
				rs *mockst.MockStore,
				wa *mockwa.MockWebAuthnConfig,
				tokenMaker *mocktk.MockMaker,
			) {
				rs.EXPECT().GetUserRegSession(gomock.Any(), sessionID).Times(1).Return(pending, nil)
				wa.EXPECT().FinishRegistration(tmpUser, *pending.SessionData, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().CreateUserWithCredentialsTx(gomock.Any(), txArg).Times(1).Return(user, nil)
				rs.EXPECT().DeleteUserRegSession(gomock.Any(), sessionID).Times(1).Return(nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("access_token", tokenPayload, nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("", &token.Payload{}, errors.New(""))
				store.EXPECT().CreateSession(gomock.Any(), gomock.Any()).Times(0)
			},
			setupRequest: addCookie,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkInternalResourceError(t, recorder)
				require.Contains(t, recorder.Header().Get("Set-Cookie"), webauthnSessionCookie+"=;")
			},
		},
		{
			name: "CreateSessionErr",
			buildStubs: func(
				store *mockdb.MockStore,
				rs *mockst.MockStore,
				wa *mockwa.MockWebAuthnConfig,
				tokenMaker *mocktk.MockMaker,
			) {
				rs.EXPECT().GetUserRegSession(gomock.Any(), sessionID).Times(1).Return(pending, nil)
				wa.EXPECT().FinishRegistration(tmpUser, *pending.SessionData, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().CreateUserWithCredentialsTx(gomock.Any(), txArg).Times(1).Return(user, nil)
				rs.EXPECT().DeleteUserRegSession(gomock.Any(), sessionID).Times(1).Return(nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("access_token", tokenPayload, nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("refresh_token", tokenPayload, nil)
				store.EXPECT().CreateSession(gomock.Any(), sessionArg).Times(1).Return(db.Session{}, pgx.ErrNoRows)
			},
			setupRequest: func(req *http.Request) {
				addCookie(req)
				req.Header.Add("User-Agent", "chrome")
				req.RemoteAddr = "198.162.0.0:12345"
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkInternalResourceError(t, recorder)
				require.Contains(t, recorder.Header().Get("Set-Cookie"), webauthnSessionCookie+"=;")
			},
		},
		{
			name: "OK",
			buildStubs: func(
				store *mockdb.MockStore,
				rs *mockst.MockStore,
				wa *mockwa.MockWebAuthnConfig,
				tokenMaker *mocktk.MockMaker,
			) {
				rs.EXPECT().GetUserRegSession(gomock.Any(), sessionID).Times(1).Return(pending, nil)
				wa.EXPECT().FinishRegistration(tmpUser, *pending.SessionData, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().CreateUserWithCredentialsTx(gomock.Any(), txArg).Times(1).Return(user, nil)
				rs.EXPECT().DeleteUserRegSession(gomock.Any(), sessionID).Times(1).Return(nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("access_token", tokenPayload, nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("refresh_token", tokenPayload, nil)
				store.EXPECT().CreateSession(gomock.Any(), sessionArg).Times(1).Return(createdSession, nil)
			},
			setupRequest: func(req *http.Request) {
				addCookie(req)
				req.Header.Add("User-Agent", "chrome")
				req.RemoteAddr = "198.162.0.0:12345"
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				require.Contains(t, recorder.Header().Get("Set-Cookie"), webauthnSessionCookie+"=;")
				require.Contains(t, recorder.Header().Get("Set-Cookie"), "Max-Age=0")

				var resp PrivateSuccessAuthResponse
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, createdSession.ID, resp.SessionID)
				require.Equal(t, "access_token", resp.AccessToken)
				require.True(t, tokenPayload.ExpiredAt.Equal(resp.AccessTokenExpiresAt))
				require.Equal(t, "refresh_token", resp.RefreshToken)
				require.True(t, tokenPayload.ExpiredAt.Equal(resp.RefreshTokenExpiresAt))
				require.Equal(t, user.ID, resp.User.ID)
				require.Equal(t, user.Username, resp.User.Username)
				require.Equal(t, user.Email, resp.User.Email)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// mock store
			dbCtrl := gomock.NewController(t)
			defer dbCtrl.Finish()
			store := mockdb.NewMockStore(dbCtrl)

			// mock redis
			rsCtrl := gomock.NewController(t)
			defer rsCtrl.Finish()
			rs := mockst.NewMockStore(rsCtrl)

			// mock webauthn-go lib
			waCtrl := gomock.NewController(t)
			defer waCtrl.Finish()
			wa := mockwa.NewMockWebAuthnConfig(waCtrl)

			// mock token maker
			tkCtrl := gomock.NewController(t)
			defer tkCtrl.Finish()
			tk := mocktk.NewMockMaker(tkCtrl)

			tc.buildStubs(store, rs, wa, tk)

			service := newTestService(t, store, tk, rs, wa)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(http.MethodPost, "/users/signup/finish", nil)
			require.NoError(t, err)

			tc.setupRequest(request)

			service.router.ServeHTTP(recorder, request)
			require.Equal(t, contentJSON, recorder.Header().Get("Content-Type"))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestExtractTransportData(t *testing.T) {
	t.Run("from credential transport", func(t *testing.T) {
		// given credential with transport values
		cred := &webauthn.Credential{
			Transport: []protocol.AuthenticatorTransport{
				protocol.USB,
				protocol.NFC,
			},
		}

		req := httptest.NewRequest("GET", "/", nil)

		got := extractTransportData(req, cred)
		require.Equal(t, []string{"usb", "nfc"}, got)
	})

	t.Run("from header", func(t *testing.T) {
		// given credential without transports
		cred := &webauthn.Credential{}

		// create gin context with header
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set(WebauthnTransportHeader, "ble,internal")

		got := extractTransportData(req, cred)
		require.Equal(t, []string{"ble", "internal"}, got)
	})

	t.Run("empty header", func(t *testing.T) {
		cred := &webauthn.Credential{}
		req := httptest.NewRequest("GET", "/", nil)

		got := extractTransportData(req, cred)
		// strings.Split("", ",") возвращает []string{""}
		require.Equal(t, []string{""}, got)
	})
}
