package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/stretchr/testify/require"
)

func TestCORSMiddleware(t *testing.T) {
	allowedOrigin := "http://localhost:3000"

	testCases := []struct {
		name                    string
		method                  string
		origin                  string
		allowedOrigins          []string
		wantStatus              int
		wantNextCalled          bool
		wantAllowOrigin         string
		wantAllowCredentials    string
		wantAllowMethods        string
		wantAllowHeaders        string
	}{
		{
			name:                 "AllowedOriginPassesThrough",
			method:               http.MethodGet,
			origin:               allowedOrigin,
			allowedOrigins:       []string{allowedOrigin},
			wantStatus:           http.StatusTeapot,
			wantNextCalled:       true,
			wantAllowOrigin:      allowedOrigin,
			wantAllowCredentials: "true",
			wantAllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
			wantAllowHeaders:     "Content-Type,Authorization," + WebauthnTransportHeader,
		},
		{
			name:           "DisallowedOriginPassesThroughWithoutCORSHeaders",
			method:         http.MethodGet,
			origin:         "http://evil.example",
			allowedOrigins: []string{allowedOrigin},
			wantStatus:     http.StatusTeapot,
			wantNextCalled: true,
		},
		{
			name:                 "AllowedPreflightStopsChain",
			method:               http.MethodOptions,
			origin:               allowedOrigin,
			allowedOrigins:       []string{allowedOrigin},
			wantStatus:           http.StatusNoContent,
			wantNextCalled:       false,
			wantAllowOrigin:      allowedOrigin,
			wantAllowCredentials: "true",
			wantAllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
			wantAllowHeaders:     "Content-Type,Authorization," + WebauthnTransportHeader,
		},
		{
			name:           "PreflightWithoutAllowedOriginStopsChainWithoutCORSHeaders",
			method:         http.MethodOptions,
			allowedOrigins: []string{allowedOrigin},
			wantStatus:     http.StatusNoContent,
			wantNextCalled: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := Service{
				config: util.Config{
					AllowedOrigins: tc.allowedOrigins,
				},
			}

			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusTeapot)
			})

			handler := service.corsMiddleware(next)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tc.method, "/users/1", nil)
			if tc.origin != "" {
				request.Header.Set("Origin", tc.origin)
			}

			handler.ServeHTTP(recorder, request)

			require.Equal(t, tc.wantStatus, recorder.Code)
			require.Equal(t, tc.wantNextCalled, nextCalled)
			require.Contains(t, recorder.Header().Values("Vary"), "Origin")
			require.Equal(t, tc.wantAllowOrigin, recorder.Header().Get("Access-Control-Allow-Origin"))
			require.Equal(t, tc.wantAllowCredentials, recorder.Header().Get("Access-Control-Allow-Credentials"))
			require.Equal(t, tc.wantAllowMethods, recorder.Header().Get("Access-Control-Allow-Methods"))
			require.Equal(t, tc.wantAllowHeaders, recorder.Header().Get("Access-Control-Allow-Headers"))
		})
	}
}
