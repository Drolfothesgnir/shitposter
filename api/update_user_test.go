package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/Drolfothesgnir/shitposter/db/mock"
	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUpdateUser(t *testing.T) {

	imgURL := util.RandomURL()
	userID := util.RandomInt(1, 1000)
	username := util.RandomOwner()
	email := util.RandomEmail()

	arg := db.UpdateUserParams{
		ID:            userID,
		Username:      &username,
		Email:         &email,
		ProfileImgURL: &imgURL,
	}

	testCases := []struct {
		name          string
		body          reqBody
		buildStubs    func(store *mockdb.MockStore)
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "InvalidUsername",
			body: reqBody{
				"username": "./1",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, userID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				var resp Vomit
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindPayload, resp.Kind)
				require.Equal(t, ReqInvalidArguments, resp.Reason)
				require.Equal(t, http.StatusBadRequest, resp.Status)
				require.Len(t, resp.Issues, 1)
				require.Equal(t, "username", resp.Issues[0].FieldName)
				require.Equal(t, "alphanum", resp.Issues[0].Tag)
			},
		},
		{
			name: "EmptyBody",
			body: reqBody{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, userID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				var resp Vomit
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindPayload, resp.Kind)
				require.Equal(t, ReqInvalidArguments, resp.Reason)
				require.Equal(t, http.StatusBadRequest, resp.Status)
				require.Equal(t, "invalid request arguments", resp.ErrMessage)
				require.Len(t, resp.Issues, 1)
				require.Equal(t, "body", resp.Issues[0].FieldName)
				require.Equal(t, "empty_body", resp.Issues[0].Tag)
				require.Equal(t, "at least one of the optional fields must be present", resp.Issues[0].Message)
			},
		},
		{
			name: "UserNotFound",
			body: reqBody{
				"username": username,
				"email":    email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), db.UpdateUserParams{
					ID:       userID,
					Username: &username,
					Email:    &email,
				}).Times(1).Return(
					db.UpdateUserResult{},
					&db.OpError{
						Op:       "update-user",
						Kind:     db.KindNotFound,
						Entity:   "user",
						EntityID: fmt.Sprint(userID),
						Err:      fmt.Errorf("user with id %d not found", userID),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, userID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, http.StatusNotFound, resp.Status)
				require.Equal(t, "not_found", resp.Reason)
				require.Equal(t, fmt.Sprintf("user with id %d not found", userID), resp.Error)
			},
		},
		{
			name: "DuplicateUsername",
			body: reqBody{
				"username":        username,
				"email":           email,
				"profile_img_url": imgURL,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), arg).Times(1).Return(
					db.UpdateUserResult{},
					&db.OpError{
						Op:           "update-user",
						Kind:         db.KindConflict,
						Entity:       "user",
						EntityID:     fmt.Sprint(userID),
						FailingField: "username",
						Err:          fmt.Errorf("user with username '%s' exists", username),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, userID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, http.StatusConflict, resp.Status)
				require.Equal(t, "conflict", resp.Reason)
				require.Equal(t, fmt.Sprintf("user with username '%s' exists", username), resp.Error)
			},
		},
		{
			name: "DuplicateEmail",
			body: reqBody{
				"username":        username,
				"email":           email,
				"profile_img_url": imgURL,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), arg).Times(1).Return(
					db.UpdateUserResult{},
					&db.OpError{
						Op:           "update-user",
						Kind:         db.KindConflict,
						Entity:       "user",
						EntityID:     fmt.Sprint(userID),
						FailingField: "email",
						Err:          fmt.Errorf("user with email '%s' exists", email),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, userID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, http.StatusConflict, resp.Status)
				require.Equal(t, "conflict", resp.Reason)
				require.Equal(t, fmt.Sprintf("user with email '%s' exists", email), resp.Error)
			},
		},
		{
			name: "UpdateUserErr",
			body: reqBody{
				"username":        username,
				"email":           email,
				"profile_img_url": imgURL,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), arg).Times(1).Return(
					db.UpdateUserResult{},
					&db.OpError{
						Op:     "update-user",
						Kind:   db.KindInternal,
						Entity: "user",
						Err:    fmt.Errorf("tx closed"),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, userID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, http.StatusInternalServerError, resp.Status)
				require.Equal(t, "internal", resp.Reason)
				require.Equal(t, "an internal error occurred", resp.Error)
			},
		},
		{
			name: "OK",
			body: reqBody{
				"username":        username,
				"email":           email,
				"profile_img_url": imgURL,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), arg).Times(1).Return(
					db.UpdateUserResult{
						ID:       userID,
						Username: username,
						Email:    email,
					},
					nil,
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, userID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var resp db.UpdateUserResult
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, userID, resp.ID)
				require.Equal(t, username, resp.Username)
				require.Equal(t, email, resp.Email)
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
			require.Equal(t, contentJSON, recorder.Header().Get("Content-Type"))
			tc.checkResponse(t, recorder)
		})
	}
}
