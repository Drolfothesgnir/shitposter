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
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSignupFinish(t *testing.T) {

	userHandle := util.RandomByteArray(32)

	aaguid := util.RandomByteArray(16)

	transports := []protocol.AuthenticatorTransport{
		protocol.USB,
		protocol.NFC,
	}

	transportsJson, err := json.Marshal(transports)
	require.NoError(t, err)

	txArg := db.CreateUserWithCredentialsTxParams{
		User: db.CreateUserParams{
			Username:           util.RandomOwner(),
			Email:              util.RandomEmail(),
			WebauthnUserHandle: userHandle,
		},
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
		ID:        uuid.UUID{byte(user.ID)},
		UserID:    user.ID,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(time.Minute),
	}

	testCases := []struct {
		name       string
		buildStubs func(
			store *mockdb.MockStore,
			rs *mockst.MockStore,
			wa *mockwa.MockWebAuthnConfig,
			tokenMaker *mocktk.MockMaker,
		)
		setupHeaders  func(req *http.Request)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "MissingHeader",
			buildStubs: func(
				store *mockdb.MockStore,
				rs *mockst.MockStore,
				wa *mockwa.MockWebAuthnConfig,
				tokenMaker *mocktk.MockMaker,
			) {
				rs.EXPECT().GetUserRegSession(gomock.Any(), gomock.Any()).Times(0)
			},
			setupHeaders: func(req *http.Request) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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
				rs.EXPECT().GetUserRegSession(gomock.Any(), "session_id").Times(1).Return(session, errors.New(""))
				wa.EXPECT().FinishRegistration(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Add(WebauthnChallengeHeader, "session_id")
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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
				rs.EXPECT().GetUserRegSession(gomock.Any(), "session_id").Times(1).Return(session, nil)
				wa.EXPECT().FinishRegistration(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Add(WebauthnChallengeHeader, "session_id")
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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
				rs.EXPECT().GetUserRegSession(gomock.Any(), "session_id").Times(1).Return(pending, nil)
				wa.EXPECT().FinishRegistration(gomock.Any(), *pending.SessionData, gomock.Any()).Times(1).Return(&webauthn.Credential{}, errors.New(""))
				store.EXPECT().CreateUserWithCredentialsTx(gomock.Any(), gomock.Any()).Times(0)
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Add(WebauthnChallengeHeader, "session_id")
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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
				rs.EXPECT().GetUserRegSession(gomock.Any(), "session_id").Times(1).Return(pending, nil)
				wa.EXPECT().FinishRegistration(tmpUser, *pending.SessionData, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().CreateUserWithCredentialsTx(gomock.Any(), txArg).Times(1).Return(db.CreateUserWithCredentialsTxResult{}, pgx.ErrTxClosed)
				rs.EXPECT().DeleteUserRegSession(gomock.Any(), "session_id").Times(0)
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Add(WebauthnChallengeHeader, "session_id")
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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
				rs.EXPECT().GetUserRegSession(gomock.Any(), "session_id").Times(1).Return(pending, nil)
				wa.EXPECT().FinishRegistration(tmpUser, *pending.SessionData, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().CreateUserWithCredentialsTx(gomock.Any(), txArg).Times(1).Return(db.CreateUserWithCredentialsTxResult{
					User: user,
				}, nil)
				rs.EXPECT().DeleteUserRegSession(gomock.Any(), "session_id").Times(1).Return(nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("", &token.Payload{}, errors.New(""))
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(0)
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Add(WebauthnChallengeHeader, "session_id")
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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
				rs.EXPECT().GetUserRegSession(gomock.Any(), "session_id").Times(1).Return(pending, nil)
				wa.EXPECT().FinishRegistration(tmpUser, *pending.SessionData, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().CreateUserWithCredentialsTx(gomock.Any(), txArg).Times(1).Return(db.CreateUserWithCredentialsTxResult{
					User: user,
				}, nil)
				rs.EXPECT().DeleteUserRegSession(gomock.Any(), "session_id").Times(1).Return(nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("access_token", tokenPayload, nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("", &token.Payload{}, errors.New(""))
				store.EXPECT().CreateSession(gomock.Any(), gomock.Any()).Times(0)
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Add(WebauthnChallengeHeader, "session_id")
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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
				rs.EXPECT().GetUserRegSession(gomock.Any(), "session_id").Times(1).Return(pending, nil)
				wa.EXPECT().FinishRegistration(tmpUser, *pending.SessionData, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().CreateUserWithCredentialsTx(gomock.Any(), txArg).Times(1).Return(db.CreateUserWithCredentialsTxResult{
					User: user,
				}, nil)
				rs.EXPECT().DeleteUserRegSession(gomock.Any(), "session_id").Times(1).Return(nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("access_token", tokenPayload, nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("refresh_token", tokenPayload, nil)
				store.EXPECT().CreateSession(gomock.Any(), gomock.Any()).Times(1).Return(db.Session{}, pgx.ErrNoRows)
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Add(WebauthnChallengeHeader, "session_id")
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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
				sessionArg := db.CreateSessionParams{
					ID:           tokenPayload.ID,
					UserID:       tokenPayload.UserID,
					RefreshToken: "refresh_token",
					UserAgent:    "chrome",
					ClientIp:     "198.162.0.0",
					IsBlocked:    false,
					ExpiresAt:    tokenPayload.ExpiredAt,
				}

				session := db.Session{
					ID:           tokenPayload.ID,
					UserID:       tokenPayload.UserID,
					RefreshToken: "refresh_token",
					UserAgent:    "chrome",
					ClientIp:     "198.162.0.0",
					IsBlocked:    false,
					ExpiresAt:    tokenPayload.ExpiredAt,
				}

				rs.EXPECT().GetUserRegSession(gomock.Any(), "session_id").Times(1).Return(pending, nil)
				wa.EXPECT().FinishRegistration(tmpUser, *pending.SessionData, gomock.Any()).Times(1).Return(waCred, nil)
				store.EXPECT().CreateUserWithCredentialsTx(gomock.Any(), txArg).Times(1).Return(db.CreateUserWithCredentialsTxResult{
					User: user,
				}, nil)
				rs.EXPECT().DeleteUserRegSession(gomock.Any(), "session_id").Times(1).Return(nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("access_token", tokenPayload, nil)
				tokenMaker.EXPECT().CreateToken(user.ID, time.Minute).Times(1).Return("refresh_token", tokenPayload, nil)
				store.EXPECT().CreateSession(gomock.Any(), sessionArg).Times(1).Return(session, nil)
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Add(WebauthnChallengeHeader, "session_id")
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

			request, err := http.NewRequest(http.MethodPost, UsersSignupFinishURL, nil)
			require.NoError(t, err)

			tc.setupHeaders(request)

			service.router.ServeHTTP(recorder, request)
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

		// gin context не нужен для этого пути, можно создать пустой
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

		got := extractTransportData(ctx, cred)
		require.Equal(t, []string{"usb", "nfc"}, got)
	})

	t.Run("from header", func(t *testing.T) {
		// given credential without transports
		cred := &webauthn.Credential{}

		// create gin context with header
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set(WebauthnTransportHeader, "ble,internal")
		ctx.Request = req

		got := extractTransportData(ctx, cred)
		require.Equal(t, []string{"ble", "internal"}, got)
	})

	t.Run("empty header", func(t *testing.T) {
		cred := &webauthn.Credential{}
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/", nil)
		ctx.Request = req

		got := extractTransportData(ctx, cred)
		// strings.Split("", ",") возвращает []string{""}
		require.Equal(t, []string{""}, got)
	})
}
