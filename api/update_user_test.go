package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/Drolfothesgnir/shitposter/db/mock"
	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/gin-gonic/gin"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUpdateUser(t *testing.T) {

	user := db.User{
		ID:            util.RandomInt(1, 1000),
		Username:      util.RandomOwner(),
		Email:         util.RandomEmail(),
		ProfileImgUrl: pgtype.Text{String: util.RandomURL(), Valid: true},
	}

	arg := db.UpdateUserParams{
		ID:            user.ID,
		Username:      util.StringToPgxText(&user.Username),
		Email:         util.StringToPgxText(&user.Email),
		ProfileImgUrl: user.ProfileImgUrl,
	}

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "InvalidUsername",
			body: gin.H{
				"username": "./1",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, "invalid params", res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, res.Fields[0].FieldName, "username")
				require.Equal(t, res.Fields[0].ErrorMessage, getBindingErrorMessage("alphanum", "./1", ""))
			},
		},
		{
			name: "EmptyBody",
			body: gin.H{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "UserNotFound",
			body: gin.H{
				"username": user.Username,
				"email":    user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.UpdateUserParams{
					ID:       user.ID,
					Username: arg.Username,
					Email:    arg.Email,
				}
				store.EXPECT().UpdateUser(gomock.Any(), arg).Times(1).Return(db.User{}, pgx.ErrNoRows)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "DuplicateUsername",
			body: gin.H{
				"username":        user.Username,
				"email":           user.Email,
				"profile_img_url": arg.ProfileImgUrl.String,
			},
			buildStubs: func(store *mockdb.MockStore) {
				err := &pgconn.PgError{
					Code:           "23505",
					ConstraintName: "uniq_users_username_active",
				}
				store.EXPECT().UpdateUser(gomock.Any(), arg).Times(1).Return(db.User{}, err)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, "username already in use", res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, res.Fields[0].FieldName, "username")
				require.Equal(t, res.Fields[0].ErrorMessage, "already in use")
			},
		},
		{
			name: "DuplicateEmail",
			body: gin.H{
				"username":        user.Username,
				"email":           user.Email,
				"profile_img_url": arg.ProfileImgUrl.String,
			},
			buildStubs: func(store *mockdb.MockStore) {
				err := &pgconn.PgError{
					Code:           "23505",
					ConstraintName: "uniq_users_email_active",
				}
				store.EXPECT().UpdateUser(gomock.Any(), arg).Times(1).Return(db.User{}, err)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, "email already in use", res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, res.Fields[0].FieldName, "email")
				require.Equal(t, res.Fields[0].ErrorMessage, "already in use")
			},
		},
		{
			name: "UpdateUserErr",
			body: gin.H{
				"username":        user.Username,
				"email":           user.Email,
				"profile_img_url": arg.ProfileImgUrl.String,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), arg).Times(1).Return(db.User{}, pgx.ErrTxClosed)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "OK",
			body: gin.H{
				"username":        user.Username,
				"email":           user.Email,
				"profile_img_url": arg.ProfileImgUrl.String,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), arg).Times(1).Return(user, nil)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
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

			tokenMaker, err := token.NewJWTMaker(testConfig.TokenSymmetricKey)
			require.NoError(t, err)

			tc.buildStubs(store)

			service := newTestService(t, store, tokenMaker, nil, nil)
			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			request, err := http.NewRequest(http.MethodPatch, "/users", bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, tokenMaker)

			service.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
