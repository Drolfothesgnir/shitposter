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

	create := &protocol.CredentialCreation{
		Response: protocol.PublicKeyCredentialCreationOptions{
			Challenge: protocol.URLEncodedBase64([]byte("signup-challenge")),
		},
	}

	session := &webauthn.SessionData{
		Challenge: "chal",
	}

	registrationData := tmpstore.PendingRegistration{
		Email:       email,
		Username:    username,
		SessionData: session,
		ExpiresAt:   time.Now().Add(testConfig.RegistrationSessionTTL),
	}

	checkInvalidArguments := func(t *testing.T, recorder *httptest.ResponseRecorder, issues ...Issue) {
		t.Helper()

		require.Equal(t, http.StatusBadRequest, recorder.Code)

		var resp Vomit
		err := json.NewDecoder(recorder.Body).Decode(&resp)
		require.NoError(t, err)
		require.Equal(t, KindPayload, resp.Kind)
		require.Equal(t, ReqInvalidArguments, resp.Reason)
		require.Equal(t, http.StatusBadRequest, resp.Status)
		require.Equal(t, "invalid request arguments", resp.ErrMessage)
		require.Equal(t, issues, resp.Issues)
	}

	checkConflictVomit := func(t *testing.T, recorder *httptest.ResponseRecorder, msg string) {
		t.Helper()

		require.Equal(t, http.StatusConflict, recorder.Code)

		var resp Vomit
		err := json.NewDecoder(recorder.Body).Decode(&resp)
		require.NoError(t, err)
		require.Equal(t, Vomit{
			Kind:       KindPayload,
			Reason:     ReqInvalidArguments,
			Status:     http.StatusConflict,
			ErrMessage: msg,
		}, resp)
	}

	checkInternalResourceError := func(t *testing.T, recorder *httptest.ResponseRecorder) {
		t.Helper()

		require.Equal(t, http.StatusInternalServerError, recorder.Code)

		var resp ResourceError
		err := json.NewDecoder(recorder.Body).Decode(&resp)
		require.NoError(t, err)
		require.Equal(t, ResourceError{
			Kind:   KindResource,
			Reason: "internal",
			Status: http.StatusInternalServerError,
			Error:  "an internal error occurred",
		}, resp)
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
				"username": ".,/",
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkInvalidArguments(t, recorder, Issue{
					FieldName: "username",
					Tag:       "alphanum",
					Message:   "value must only letters and numbers",
				})
			},
		},
		{
			name: "InvalidEmail",
			body: reqBody{
				"username": username,
				"email":    "123",
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkInvalidArguments(t, recorder, Issue{
					FieldName: "email",
					Tag:       "email",
					Message:   "field must be a correct email address",
				})
			},
		},
		{
			name: "UsernameExistsErr",
			body: reqBody{
				"username": username,
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), username).Times(1).Return(false, pgx.ErrTxClosed)
				store.EXPECT().EmailExists(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkInternalResourceError(t, recorder)
			},
		},
		{
			name: "UsernameExists",
			body: reqBody{
				"username": username,
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), username).Times(1).Return(true, nil)
				store.EXPECT().EmailExists(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkConflictVomit(t, recorder, "user with username ["+username+"] already exists")
			},
		},
		{
			name: "EmailExistsErr",
			body: reqBody{
				"username": username,
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), username).Times(1).Return(false, nil)
				store.EXPECT().EmailExists(gomock.Any(), email).Times(1).Return(false, pgx.ErrTxClosed)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkInternalResourceError(t, recorder)
			},
		},
		{
			name: "EmailExists",
			body: reqBody{
				"username": username,
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				store.EXPECT().UsernameExists(gomock.Any(), username).Times(1).Return(false, nil)
				store.EXPECT().EmailExists(gomock.Any(), email).Times(1).Return(true, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkConflictVomit(t, recorder, "user with email ["+email+"] already exists")
			},
		},
		{
			name: "BeginRegistrationErr",
			body: reqBody{
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
					require.Len(t, user.WebauthnUserHandle, 32)
					require.Equal(t, user.ID, user.WebauthnUserHandle)
					return &protocol.CredentialCreation{}, &webauthn.SessionData{}, errors.New("")
				})
				rs.EXPECT().SaveUserRegSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkInternalResourceError(t, recorder)
			},
		},
		{
			name: "SaveRegistrationErr",
			body: reqBody{
				"username": username,
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				var expectedHandle []byte

				store.EXPECT().UsernameExists(gomock.Any(), username).Times(1).Return(false, nil)
				store.EXPECT().EmailExists(gomock.Any(), email).Times(1).Return(false, nil)

				wa.EXPECT().
					BeginRegistration(
						gomock.AssignableToTypeOf(&TempUser{}),
					).DoAndReturn(func(user *TempUser, _ ...[]webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
					require.Equal(t, tmpUser.Username, user.Username)
					require.Equal(t, tmpUser.Email, user.Email)
					require.Len(t, user.WebauthnUserHandle, 32)
					require.Equal(t, user.ID, user.WebauthnUserHandle)
					expectedHandle = append([]byte(nil), user.WebauthnUserHandle...)
					return create, session, nil
				})
				rs.EXPECT().
					SaveUserRegSession(
						gomock.Any(),
						gomock.Any(),
						gomock.AssignableToTypeOf(tmpstore.PendingRegistration{}),
						testConfig.RegistrationSessionTTL,
					).
					DoAndReturn(func(_ context.Context, sessionID string, pending tmpstore.PendingRegistration, ttl time.Duration) error {
						require.NotEmpty(t, sessionID)
						require.Equal(t, registrationData.Email, pending.Email)
						require.Equal(t, registrationData.Username, pending.Username)
						require.Equal(t, expectedHandle, pending.WebauthnUserHandle)
						require.Equal(t, registrationData.SessionData, pending.SessionData)
						require.WithinDuration(t, pending.ExpiresAt, registrationData.ExpiresAt, time.Second)
						return errors.New("")
					})
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				checkInternalResourceError(t, recorder)
			},
		},
		{
			name: "OK",
			body: reqBody{
				"username": username,
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {
				var expectedHandle []byte

				store.EXPECT().UsernameExists(gomock.Any(), username).Times(1).Return(false, nil)
				store.EXPECT().EmailExists(gomock.Any(), email).Times(1).Return(false, nil)

				wa.EXPECT().
					BeginRegistration(
						gomock.AssignableToTypeOf(&TempUser{}),
					).DoAndReturn(func(user *TempUser, _ ...[]webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
					require.Equal(t, tmpUser.Username, user.Username)
					require.Equal(t, tmpUser.Email, user.Email)
					require.Len(t, user.WebauthnUserHandle, 32)
					require.Equal(t, user.ID, user.WebauthnUserHandle)
					expectedHandle = append([]byte(nil), user.WebauthnUserHandle...)
					return create, session, nil
				})
				rs.EXPECT().SaveUserRegSession(
					gomock.Any(),
					gomock.Any(),
					gomock.AssignableToTypeOf(tmpstore.PendingRegistration{}),
					testConfig.RegistrationSessionTTL,
				).
					DoAndReturn(func(_ context.Context, sessionID string, pending tmpstore.PendingRegistration, ttl time.Duration) error {
						require.NotEmpty(t, sessionID)
						require.Equal(t, registrationData.Email, pending.Email)
						require.Equal(t, registrationData.Username, pending.Username)
						require.Equal(t, expectedHandle, pending.WebauthnUserHandle)
						require.Equal(t, registrationData.SessionData, pending.SessionData)
						require.WithinDuration(t, pending.ExpiresAt, registrationData.ExpiresAt, time.Second)
						return nil
					})
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				require.Contains(t, recorder.Header().Get("Set-Cookie"), webauthnSessionCookie+"=")
				require.Contains(t, recorder.Header().Get("Set-Cookie"), "Max-Age=60")
				require.Contains(t, recorder.Header().Get("Set-Cookie"), "HttpOnly")
				require.Contains(t, recorder.Header().Get("Set-Cookie"), "Secure")

				var resp SignupStartResponse
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.NotNil(t, resp.CredentialCreation)
				require.True(t, assertionEqualCreate(create, resp.CredentialCreation))
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

			request, err := http.NewRequest(http.MethodPost, "/users/signup/start", bytes.NewReader(data))
			require.NoError(t, err)

			service.router.ServeHTTP(recorder, request)
			require.Equal(t, contentJSON, recorder.Header().Get("Content-Type"))
			tc.checkResponse(t, recorder)
		})
	}
}

func assertionEqualCreate(expected, actual *protocol.CredentialCreation) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	return bytes.Equal(expected.Response.Challenge, actual.Response.Challenge)
}
