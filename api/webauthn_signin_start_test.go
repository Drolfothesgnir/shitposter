package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
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

	transportsJSON, err := json.Marshal(transports)
	require.NoError(t, err)

	cred := db.WebauthnCredential{
		ID:         userHandle,
		UserID:     user.ID,
		Transports: transportsJSON,
	}

	userWithCreds, err := NewUserWithCredentials(user, []db.WebauthnCredential{cred})
	require.NoError(t, err)

	assertion := &protocol.CredentialAssertion{
		Response: protocol.PublicKeyCredentialRequestOptions{
			Challenge: protocol.URLEncodedBase64([]byte("challenge")),
			AllowedCredentials: []protocol.CredentialDescriptor{{
				Type:         protocol.PublicKeyCredentialType,
				CredentialID: protocol.URLEncodedBase64(cred.ID),
				Transport:    transports,
			}},
		},
	}
	session := &webauthn.SessionData{
		Challenge: "challenge",
	}

	checkInvalidArguments := func(t *testing.T, recorder *httptest.ResponseRecorder, fieldName, tag string) {
		t.Helper()

		require.Equal(t, http.StatusBadRequest, recorder.Code)

		var resp Vomit
		err := json.NewDecoder(recorder.Body).Decode(&resp)
		require.NoError(t, err)
		require.Equal(t, KindPayload, resp.Kind)
		require.Equal(t, ReqInvalidArguments, resp.Reason)
		require.Equal(t, http.StatusBadRequest, resp.Status)
		require.Equal(t, "invalid request arguments", resp.ErrMessage)
		require.Len(t, resp.Issues, 1)
		require.Equal(t, fieldName, resp.Issues[0].FieldName)
		require.Equal(t, tag, resp.Issues[0].Tag)
	}

	checkResourceError := func(t *testing.T, recorder *httptest.ResponseRecorder, expected ResourceError) {
		t.Helper()

		require.Equal(t, expected.Status, recorder.Code)

		var resp ResourceError
		err := json.NewDecoder(recorder.Body).Decode(&resp)
		require.NoError(t, err)
		require.Equal(t, expected, resp)
	}

	checkInternalResourceError := func(t *testing.T, recorder *httptest.ResponseRecorder) {
		t.Helper()

		checkResourceError(t, recorder, ResourceError{
			Kind:   KindResource,
			Reason: db.KindInternal.String(),
			Status: http.StatusInternalServerError,
			Error:  "an internal error occurred",
		})
	}

	testCases := []struct {
		name          string
		body          reqBody
		buildStubs    func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "InvalidUsername",
			body: reqBody{
				"username": "./-",
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().GetUserByUsername(gomock.Any(), "./-").Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkInvalidArguments(t, recorder, "username", "alphanum")
			},
		},
		{
			name: "UserNotFound",
			body: reqBody{
				"username": username,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().GetUserByUsername(gomock.Any(), username).Times(1).Return(
					db.User{},
					&db.OpError{
						Op:     "get-user-by-username",
						Kind:   db.KindNotFound,
						Entity: "user",
						Err:    fmt.Errorf("user with username %q not found", username),
					},
				)
				store.EXPECT().GetUserCredentials(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkResourceError(t, recorder, ResourceError{
					Kind:   KindResource,
					Reason: db.KindNotFound.String(),
					Status: http.StatusNotFound,
					Error:  fmt.Sprintf("user with username %q not found", username),
				})
			},
		},
		{
			name: "GetUserErr",
			body: reqBody{
				"username": username,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().GetUserByUsername(gomock.Any(), username).Times(1).Return(
					db.User{},
					&db.OpError{
						Op:     "get-user-by-username",
						Kind:   db.KindInternal,
						Entity: "user",
						Err:    fmt.Errorf("tx closed"),
					},
				)
				store.EXPECT().GetUserCredentials(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkInternalResourceError(t, recorder)
			},
		},
		{
			name: "GetUserCredsErr",
			body: reqBody{
				"username": username,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().GetUserByUsername(gomock.Any(), username).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return(
					[]db.WebauthnCredential{},
					&db.OpError{
						Op:     "get-user-credentials",
						Kind:   db.KindNotFound,
						Entity: "webauthn-credential",
						Err:    fmt.Errorf("webauthn credentials for user %d not found", user.ID),
					},
				)
				wa.EXPECT().BeginLogin(gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkResourceError(t, recorder, ResourceError{
					Kind:   KindResource,
					Reason: db.KindNotFound.String(),
					Status: http.StatusNotFound,
					Error:  fmt.Sprintf("webauthn credentials for user %d not found", user.ID),
				})
			},
		},
		{
			name: "BeginLoginErr",
			body: reqBody{
				"username": username,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().GetUserByUsername(gomock.Any(), username).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{cred}, nil)
				wa.EXPECT().BeginLogin(userWithCreds).Times(1).Return(assertion, &webauthn.SessionData{}, errors.New(""))
				rs.EXPECT().SaveUserAuthSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkInternalResourceError(t, recorder)
			},
		},
		{
			name: "SaveSessionErr",
			body: reqBody{
				"username": username,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().GetUserByUsername(gomock.Any(), username).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{cred}, nil)
				wa.EXPECT().BeginLogin(userWithCreds).Times(1).Return(assertion, session, nil)
				rs.EXPECT().
					SaveUserAuthSession(
						gomock.Any(),
						gomock.Any(),
						gomock.AssignableToTypeOf(tmpstore.PendingAuthentication{}),
						time.Minute,
					).
					DoAndReturn(
						func(_ context.Context, sessionID string, pa tmpstore.PendingAuthentication, ttl time.Duration) error {
							require.NotEmpty(t, sessionID)
							require.Equal(t, username, pa.Username)
							require.Equal(t, user.ID, pa.UserID)
							require.Same(t, session, pa.SessionData)
							require.WithinDuration(t, time.Now().Add(ttl), pa.ExpiresAt, time.Second)
							return errors.New("")
						},
					)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkInternalResourceError(t, recorder)
				require.Empty(t, recorder.Header().Get("Set-Cookie"))
			},
		},
		{
			name: "OK",
			body: reqBody{
				"username": username,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().GetUserByUsername(gomock.Any(), username).Times(1).Return(user, nil)
				store.EXPECT().GetUserCredentials(gomock.Any(), user.ID).Times(1).Return([]db.WebauthnCredential{cred}, nil)
				wa.EXPECT().BeginLogin(userWithCreds).Times(1).Return(assertion, session, nil)
				rs.EXPECT().
					SaveUserAuthSession(
						gomock.Any(),
						gomock.Any(),
						gomock.AssignableToTypeOf(tmpstore.PendingAuthentication{}),
						time.Minute,
					).
					DoAndReturn(
						func(_ context.Context, sessionID string, pa tmpstore.PendingAuthentication, ttl time.Duration) error {
							require.NotEmpty(t, sessionID)
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
				require.Contains(t, recorder.Header().Get("Set-Cookie"), webauthnSessionCookie+"=")
				require.Contains(t, recorder.Header().Get("Set-Cookie"), "Max-Age=60")
				require.Contains(t, recorder.Header().Get("Set-Cookie"), "HttpOnly")
				require.Contains(t, recorder.Header().Get("Set-Cookie"), "Secure")

				var resp SigninStartResponse
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.NotNil(t, resp.CredentialAssertion)
				require.Equal(t, assertion.Response.Challenge, resp.CredentialAssertion.Response.Challenge)
				require.Len(t, resp.CredentialAssertion.Response.AllowedCredentials, 1)
				require.Equal(t, assertion.Response.AllowedCredentials[0].Type, resp.CredentialAssertion.Response.AllowedCredentials[0].Type)
				require.Equal(t, assertion.Response.AllowedCredentials[0].CredentialID, resp.CredentialAssertion.Response.AllowedCredentials[0].CredentialID)
				require.Equal(t, assertion.Response.AllowedCredentials[0].Transport, resp.CredentialAssertion.Response.AllowedCredentials[0].Transport)
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

			service := newTestService(t, store, nil, rs, wa)
			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			request, err := http.NewRequest(http.MethodPost, "/users/signin/start", bytes.NewReader(data))
			require.NoError(t, err)

			service.router.ServeHTTP(recorder, request)
			require.Equal(t, contentJSON, recorder.Header().Get("Content-Type"))
			tc.checkResponse(t, recorder)
		})
	}
}
