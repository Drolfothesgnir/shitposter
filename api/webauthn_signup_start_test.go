package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/Drolfothesgnir/shitposter/db/mock"
	mockst "github.com/Drolfothesgnir/shitposter/tmpstore/mock"
	mockwa "github.com/Drolfothesgnir/shitposter/wauthn/mock"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSignupStart(t *testing.T) {
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
				"email":    "test@mail.com",
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
				"username": "test",
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
				"username": "test",
				"email":    "test@mail.com",
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), gomock.Any()).Times(1).Return(false, pgx.ErrTxClosed)
				store.EXPECT().EmailExists(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "UsernameExists",
			body: gin.H{
				"username": "test",
				"email":    "test@mail.com",
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), gomock.Any()).Times(1).Return(true, nil)
				store.EXPECT().EmailExists(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "EmailExistsErr",
			body: gin.H{
				"username": "test",
				"email":    "test@mail.com",
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), gomock.Any()).Times(1).Return(false, nil)
				store.EXPECT().EmailExists(gomock.Any(), gomock.Any()).Times(1).Return(false, pgx.ErrTxClosed)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "EmailExists",
			body: gin.H{
				"username": "test",
				"email":    "test@mail.com",
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), gomock.Any()).Times(1).Return(false, nil)
				store.EXPECT().EmailExists(gomock.Any(), gomock.Any()).Times(1).Return(true, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "BeginRegistrationErr",
			body: gin.H{
				"username": "test",
				"email":    "test@mail.com",
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), gomock.Any()).Times(1).Return(false, nil)
				store.EXPECT().EmailExists(gomock.Any(), gomock.Any()).Times(1).Return(false, nil)

				var create protocol.CredentialCreation
				var session webauthn.SessionData

				wa.EXPECT().BeginRegistration(gomock.Any(), gomock.Any()).Times(1).Return(&create, &session, errors.New(""))
				rs.EXPECT().SaveUserRegSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "SaveRegistrationErr",
			body: gin.H{
				"username": "test",
				"email":    "test@mail.com",
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), gomock.Any()).Times(1).Return(false, nil)
				store.EXPECT().EmailExists(gomock.Any(), gomock.Any()).Times(1).Return(false, nil)

				var create protocol.CredentialCreation
				var session webauthn.SessionData

				wa.EXPECT().BeginRegistration(gomock.Any(), gomock.Any()).Times(1).Return(&create, &session, nil)
				rs.EXPECT().SaveUserRegSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(errors.New(""))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "OK",
			body: gin.H{
				"username": "test",
				"email":    "test@mail.com",
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), gomock.Any()).Times(1).Return(false, nil)
				store.EXPECT().EmailExists(gomock.Any(), gomock.Any()).Times(1).Return(false, nil)

				var create protocol.CredentialCreation
				var session webauthn.SessionData

				wa.EXPECT().BeginRegistration(gomock.Any(), gomock.Any()).Times(1).Return(&create, &session, nil)
				rs.EXPECT().SaveUserRegSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
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

			tc.buildStubs(store, rs, wa)

			service := newTestService(t, store, rs, wa)
			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/signup/start"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			service.server.Handler.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
