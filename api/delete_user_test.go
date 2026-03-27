package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/Drolfothesgnir/shitposter/db/mock"
	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/token"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSoftDeleteUser(t *testing.T) {
	user := db.User{
		ID: 1,
	}

	testCases := []struct {
		name          string
		buildStubs    func(store *mockdb.MockStore)
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "UserNotFound",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().SoftDeleteUserTx(gomock.Any(), user.ID).Times(1).Return(
					db.SoftDeleteUserTxResult{},
					&db.OpError{
						Op:       "soft-delete-user",
						Kind:     db.KindNotFound,
						Entity:   "user",
						EntityID: fmt.Sprint(user.ID),
						Err:      fmt.Errorf("user with id %d not found", user.ID),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(
					t,
					tokenMaker,
					authorizationTypeBearer,
					user.ID,
					testConfig.AccessTokenDuration,
					request,
				)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, db.KindNotFound.String(), resp.Reason)
				require.Equal(t, fmt.Sprintf("user with id %d not found", user.ID), resp.Error)
			},
		},
		{
			name: "SoftDeleteUserTxErr",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().SoftDeleteUserTx(gomock.Any(), user.ID).Times(1).Return(
					db.SoftDeleteUserTxResult{},
					&db.OpError{
						Op:     "soft-delete-user",
						Kind:   db.KindInternal,
						Entity: "user",
						Err:    fmt.Errorf("tx closed"),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(
					t,
					tokenMaker,
					authorizationTypeBearer,
					user.ID,
					testConfig.AccessTokenDuration,
					request,
				)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, db.KindInternal.String(), resp.Reason)
				require.Equal(t, "an internal error occurred", resp.Error)
			},
		},
		{
			name: "OK",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().SoftDeleteUserTx(gomock.Any(), user.ID).Times(1).Return(
					db.SoftDeleteUserTxResult{
						ID:        user.ID,
						Username:  "deleted",
						Email:     "deleted@example.com",
						IsDeleted: true,
					},
					nil,
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(
					t,
					tokenMaker,
					authorizationTypeBearer,
					user.ID,
					testConfig.AccessTokenDuration,
					request,
				)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNoContent, recorder.Code)
				require.Empty(t, recorder.Body.String())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// mock store
			dbCtrl := gomock.NewController(t)
			defer dbCtrl.Finish()
			store := mockdb.NewMockStore(dbCtrl)

			tokenMaker, err := token.NewJWTMaker(testConfig.TokenSymmetricKey)
			require.NoError(t, err)

			tc.buildStubs(store)

			service := newTestService(t, store, tokenMaker, nil, nil)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(http.MethodDelete, "/users", nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, tokenMaker)

			service.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
