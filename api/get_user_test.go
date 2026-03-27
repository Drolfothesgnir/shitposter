package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/Drolfothesgnir/shitposter/db/mock"
	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetUser(t *testing.T) {
	user := db.User{
		ID: 1,
	}

	testCases := []struct {
		name          string
		id            string
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "InvalidParam",
			id:   "12s",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidParamNegativeInt",
			id:   "-1",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "UserNotFound",
			id:   "1",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(
					db.User{},
					&db.OpError{
						Op:       "get-user",
						Kind:     db.KindNotFound,
						Entity:   "user",
						EntityID: fmt.Sprint(user.ID),
						Err:      fmt.Errorf("user with id %d not found", user.ID),
					},
				)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, db.KindNotFound.String(), resp.Reason)
			},
		},
		{
			name: "GetUserErr",
			id:   "1",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(
					db.User{},
					&db.OpError{
						Op:     "get-user",
						Kind:   db.KindInternal,
						Entity: "user",
						Err:    fmt.Errorf("tx closed"),
					},
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
			name: "UserDeleted",
			id:   "1",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(
					db.User{},
					&db.OpError{
						Op:       "get-user",
						Kind:     db.KindDeleted,
						Entity:   "user",
						EntityID: fmt.Sprint(user.ID),
						Err:      fmt.Errorf("user with id %d is deleted", user.ID),
					},
				)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, db.KindNotFound.String(), resp.Reason)
				require.Equal(t, fmt.Sprintf("user with id [%d] not found", user.ID), resp.Error)
			},
		},
		{
			name: "OK",
			id:   "1",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), user.ID).Times(1).Return(user, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var resp PublicUserResponse
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, user.ID, resp.ID)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// mock store
			dbCtrl := gomock.NewController(t)
			defer dbCtrl.Finish()
			store := mockdb.NewMockStore(dbCtrl)

			tc.buildStubs(store)

			url := "/users/" + tc.id

			service := newTestService(t, store, nil, nil, nil)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			service.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
