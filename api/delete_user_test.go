package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/Drolfothesgnir/shitposter/db/mock"
	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/token"

	"github.com/jackc/pgx/v5"
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
			name: "NoAuthPayload",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "UserNotFound",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(db.User{}, pgx.ErrNoRows)
				store.EXPECT().SoftDeleteUserTx(gomock.Any(), gomock.Any()).Times(0)
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
			},
		},
		{
			name: "GetUserErr",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(db.User{}, pgx.ErrTxClosed)
				store.EXPECT().SoftDeleteUserTx(gomock.Any(), gomock.Any()).Times(0)
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
			},
		},
		{
			name: "DeletedUser",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(db.User{
					IsDeleted: true,
				}, nil)
				store.EXPECT().SoftDeleteUserTx(gomock.Any(), gomock.Any()).Times(0)
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
			},
		},
		{
			name: "SoftDeleteUserTxErr",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(user, nil)
				store.EXPECT().SoftDeleteUserTx(gomock.Any(), user.ID).Times(1).Return(pgx.ErrTxClosed)
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
			},
		},
		{
			name: "OK",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(user, nil)
				store.EXPECT().SoftDeleteUserTx(gomock.Any(), user.ID).Times(1).Return(nil)
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

			request, err := http.NewRequest(http.MethodDelete, UsersDeleteUser, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, tokenMaker)

			service.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
