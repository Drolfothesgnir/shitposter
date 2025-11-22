package api

import (
	"bytes"
	"context"
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
	"github.com/Drolfothesgnir/shitposter/util"
	mockwa "github.com/Drolfothesgnir/shitposter/wauthn/mock"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSignInStart(t *testing.T) {

	username := "user1"
	userHandle := util.RandomByteArray(32)
	user := db.User{
		ID:                 1,
		Username:           username,
		WebauthnUserHandle: userHandle,
		Email:              util.RandomEmail(),
	}

	transports := []protocol.AuthenticatorTransport{
		protocol.USB,
		protocol.NFC,
	}

	transportsJson, err := json.Marshal(transports)
	require.NoError(t, err)

	cred := db.WebauthnCredential{
		ID:         userHandle,
		UserID:     user.ID,
		Transports: transportsJson,
	}

	userWithCreds, err := NewUserWithCredentials(user, []db.WebauthnCredential{cred})
	require.NoError(t, err)

	session := &webauthn.SessionData{
		Challenge: "challenge",
	}

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "InvalidUsername",
			body: gin.H{
				"username": "./-",
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().GetUserByUsername(gomock.Any(), "./-").Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "UserNotFound",
			body: gin.H{
				"username": username,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().GetUserByUsername(gomock.Any(), username).Times(1).Return(db.User{}, pgx.ErrNoRows)
				store.EXPECT().GetUserCredentials(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "GetUserErr",
			body: gin.H{
				"username": username,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().GetUserByUsername(gomock.Any(), username).Times(1).Return(db.User{}, pgx.ErrTxClosed)
				store.EXPECT().GetUserCredentials(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "GetUserCredsErr",
			body: gin.H{
				"username": username,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().GetUserByUsername(gomock.Any(), username).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{}, pgx.ErrNoRows)
				wa.EXPECT().BeginLogin(gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "BeginLoginErr",
			body: gin.H{
				"username": username,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().GetUserByUsername(gomock.Any(), username).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{cred}, nil)
				wa.EXPECT().BeginLogin(userWithCreds).Times(1).Return(&protocol.CredentialAssertion{}, &webauthn.SessionData{}, errors.New(""))
				rs.EXPECT().SaveUserAuthSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "SaveSessionErr",
			body: gin.H{
				"username": username,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().GetUserByUsername(gomock.Any(), username).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{cred}, nil)
				wa.EXPECT().BeginLogin(userWithCreds).Times(1).Return(&protocol.CredentialAssertion{}, session, nil)
				rs.EXPECT().
					SaveUserAuthSession(
						gomock.Any(),
						session.Challenge,
						gomock.AssignableToTypeOf(tmpstore.PendingAuthentication{}),
						time.Minute,
					).
					DoAndReturn(
						func(_ context.Context, chal string, pa tmpstore.PendingAuthentication, ttl time.Duration) error {
							require.Equal(t, username, pa.Username)
							require.Equal(t, user.ID, pa.UserID)
							require.Same(t, session, pa.SessionData)

							require.WithinDuration(t, time.Now().Add(ttl), pa.ExpiresAt, time.Second)

							return errors.New("")
						},
					)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "SaveSessionErr",
			body: gin.H{
				"username": username,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().GetUserByUsername(gomock.Any(), username).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{cred}, nil)
				wa.EXPECT().BeginLogin(userWithCreds).Times(1).Return(&protocol.CredentialAssertion{}, session, nil)
				rs.EXPECT().
					SaveUserAuthSession(
						gomock.Any(),
						session.Challenge,
						gomock.AssignableToTypeOf(tmpstore.PendingAuthentication{}),
						time.Minute,
					).
					DoAndReturn(
						func(_ context.Context, chal string, pa tmpstore.PendingAuthentication, ttl time.Duration) error {
							require.Equal(t, username, pa.Username)
							require.Equal(t, user.ID, pa.UserID)
							require.Same(t, session, pa.SessionData)

							require.WithinDuration(t, time.Now().Add(ttl), pa.ExpiresAt, time.Second)

							return nil
						},
					)
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

			tc.buildStubs(store, rs, wa)

			service := newTestService(t, store, nil, rs, wa)
			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			request, err := http.NewRequest(http.MethodPost, "/users/signin/start", bytes.NewReader(data))
			require.NoError(t, err)

			service.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
