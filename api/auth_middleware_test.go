package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	userId := util.RandomInt(1, 1000)

	testCases := []struct {
		name               string
		setupAuth          func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		expectedStatus     int
		expectedAuthErr    *AuthError
		expectedPayloadUID int64
	}{{
		name: "OK",
		setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, userId, time.Minute, request)
		},
		expectedStatus:     http.StatusOK,
		expectedPayloadUID: userId,
	},
		{
			name: "NoAuthorization",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

			},
			expectedStatus: http.StatusUnauthorized,
			expectedAuthErr: &AuthError{
				Kind:       KindAuth,
				Reason:     AuthHeaderNotProvided,
				Status:     http.StatusUnauthorized,
				ErrMessage: "authorization header is not provided",
			},
		},
		{
			name: "InvalidHeaderFormat",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, "", userId, time.Minute, request)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedAuthErr: &AuthError{
				Kind:       KindAuth,
				Reason:     AuthInvalidHeaderFormat,
				Status:     http.StatusUnauthorized,
				ErrMessage: "invalid authorization header format",
			},
		},
		{
			name: "UnsupportedAuthorizationType",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, "unsupported", userId, time.Minute, request)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedAuthErr: &AuthError{
				Kind:       KindAuth,
				Reason:     AuthTypeUnsupported,
				Status:     http.StatusUnauthorized,
				ErrMessage: "unsupported authorization type: unsupported",
			},
		},
		{
			name: "ExpiredToken",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, userId, -time.Minute, request)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedAuthErr: &AuthError{
				Kind:       KindAuth,
				Reason:     AuthAccessTokenErr,
				Status:     http.StatusUnauthorized,
				ErrMessage: "invalid or expired token",
			},
		}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokenMaker, err := token.NewJWTMaker(testConfig.TokenSymmetricKey)
			require.NoError(t, err)

			service := newTestService(t, nil, tokenMaker, nil, nil)

			authPath := "/auth"

			testRouter := http.NewServeMux()
			nextCalled := false

			okFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true

				payload := getAuthPayload(r.Context())
				require.NotNil(t, payload)
				require.Equal(t, userId, payload.UserID)

				respondWithJSON(w, http.StatusOK, struct {
					UserID int64 `json:"user_id"`
				}{
					UserID: payload.UserID,
				})
			})

			testRouter.HandleFunc(fmt.Sprintf("GET %s", authPath), service.authMiddleware(okFn))

			service.router = testRouter

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, authPath, nil)
			require.NoError(t, err)
			tc.setupAuth(t, request, service.tokenMaker)

			service.router.ServeHTTP(recorder, request)
			require.Equal(t, tc.expectedStatus, recorder.Code)
			require.Equal(t, contentJSON, recorder.Header().Get("Content-Type"))

			if tc.expectedAuthErr != nil {
				require.False(t, nextCalled)

				var resp AuthError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, *tc.expectedAuthErr, resp)
				return
			}

			require.True(t, nextCalled)

			var resp struct {
				UserID int64 `json:"user_id"`
			}
			err = json.NewDecoder(recorder.Body).Decode(&resp)
			require.NoError(t, err)
			require.Equal(t, tc.expectedPayloadUID, resp.UserID)
		})
	}
}
