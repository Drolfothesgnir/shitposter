package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/Drolfothesgnir/shitposter/db/mock"
	mockst "github.com/Drolfothesgnir/shitposter/tmpstore/mock"
	mockwa "github.com/Drolfothesgnir/shitposter/wauthn/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSignupFinish(t *testing.T) {
	testCases := []struct {
		name          string
		buildStubs    func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig)
		setupHeaders  func(req *http.Request)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:         "MissingHeader",
			buildStubs:   func(store *mockdb.MockStore, rs *mockst.MockStore, wa *mockwa.MockWebAuthnConfig) {},
			setupHeaders: func(req *http.Request) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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

			url := "/signup/finish"
			request, err := http.NewRequest(http.MethodPost, url, nil)
			require.NoError(t, err)

			tc.setupHeaders(request)

			service.server.Handler.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
