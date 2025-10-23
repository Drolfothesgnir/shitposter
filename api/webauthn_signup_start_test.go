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

func TestSignupStart(t *testing.T) {
	username := util.RandomOwner()
	email := util.RandomEmail()

	tmpUser := &TempUser{
		Email:    email,
		Username: username,
	}

	create := &protocol.CredentialCreation{}

	session := &webauthn.SessionData{
		Challenge: "chal",
	}

	registrationData := tmpstore.PendingRegistration{
		Email:       email,
		Username:    username,
		SessionData: session,
		ExpiresAt:   time.Now().Add(testConfig.RegistrationSessionTTL),
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
				"username": ".,/",
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidEmail",
			body: gin.H{
				"username": username,
				"email":    "123",
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "UsernameExistsErr",
			body: gin.H{
				"username": username,
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), username).Times(1).Return(false, pgx.ErrTxClosed)
				store.EXPECT().EmailExists(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "UsernameExists",
			body: gin.H{
				"username": username,
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), username).Times(1).Return(true, nil)
				store.EXPECT().EmailExists(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "EmailExistsErr",
			body: gin.H{
				"username": username,
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), username).Times(1).Return(false, nil)
				store.EXPECT().EmailExists(gomock.Any(), email).Times(1).Return(false, pgx.ErrTxClosed)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "EmailExists",
			body: gin.H{
				"username": username,
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), username).Times(1).Return(false, nil)
				store.EXPECT().EmailExists(gomock.Any(), email).Times(1).Return(true, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "BeginRegistrationErr",
			body: gin.H{
				"username": username,
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), username).Times(1).Return(false, nil)
				store.EXPECT().EmailExists(gomock.Any(), email).Times(1).Return(false, nil)

				wa.EXPECT().
					BeginRegistration(
						gomock.AssignableToTypeOf(&TempUser{}),
					).DoAndReturn(func(user *TempUser, _ ...[]webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
					require.Equal(t, tmpUser.Username, user.Username)
					require.Equal(t, tmpUser.Email, user.Email)
					return &protocol.CredentialCreation{}, &webauthn.SessionData{}, errors.New("")
				})
				rs.EXPECT().SaveUserRegSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "SaveRegistrationErr",
			body: gin.H{
				"username": username,
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), username).Times(1).Return(false, nil)
				store.EXPECT().EmailExists(gomock.Any(), email).Times(1).Return(false, nil)

				wa.EXPECT().
					BeginRegistration(
						gomock.AssignableToTypeOf(&TempUser{}),
					).DoAndReturn(func(user *TempUser, _ ...[]webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
					require.Equal(t, tmpUser.Username, user.Username)
					require.Equal(t, tmpUser.Email, user.Email)
					return create, session, nil
				})
				rs.EXPECT().
					SaveUserRegSession(
						gomock.Any(),
						session.Challenge,
						gomock.AssignableToTypeOf(tmpstore.PendingRegistration{}),
						testConfig.RegistrationSessionTTL,
					).
					DoAndReturn(func(_ context.Context, chal string, pending tmpstore.PendingRegistration, ttl time.Duration) error {
						require.Equal(t, session.Challenge, chal)
						require.Equal(t, registrationData.Email, pending.Email)
						require.Equal(t, registrationData.Username, pending.Username)
						require.Equal(t, registrationData.SessionData, pending.SessionData)
						require.WithinDuration(t, pending.ExpiresAt, registrationData.ExpiresAt, time.Second)
						return errors.New("")
					})
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "OK",
			body: gin.H{
				"username": username,
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), username).Times(1).Return(false, nil)
				store.EXPECT().EmailExists(gomock.Any(), email).Times(1).Return(false, nil)

				wa.EXPECT().
					BeginRegistration(
						gomock.AssignableToTypeOf(&TempUser{}),
					).DoAndReturn(func(user *TempUser, _ ...[]webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
					require.Equal(t, tmpUser.Username, user.Username)
					require.Equal(t, tmpUser.Email, user.Email)
					return create, session, nil
				})
				rs.EXPECT().SaveUserRegSession(
					gomock.Any(),
					session.Challenge,
					gomock.AssignableToTypeOf(tmpstore.PendingRegistration{}),
					testConfig.RegistrationSessionTTL,
				).
					DoAndReturn(func(_ context.Context, chal string, pending tmpstore.PendingRegistration, ttl time.Duration) error {
						require.Equal(t, session.Challenge, chal)
						require.Equal(t, registrationData.Email, pending.Email)
						require.Equal(t, registrationData.Username, pending.Username)
						require.Equal(t, registrationData.SessionData, pending.SessionData)
						require.WithinDuration(t, pending.ExpiresAt, registrationData.ExpiresAt, time.Second)
						return nil
					})
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

			request, err := http.NewRequest(http.MethodPost, UsersSignupStartURL, bytes.NewReader(data))
			require.NoError(t, err)

			service.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
