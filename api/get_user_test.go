package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/Drolfothesgnir/shitposter/db/mock"
	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetUser(t *testing.T) {
	user := db.User{
		ID:          1,
		Username:    "alice",
		Email:       "alice@example.com",
		DisplayName: "Alice",
		CreatedAt:   time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC),
		ProfileImgUrl: pgtype.Text{
			String: "https://example.com/alice.png",
			Valid:  true,
		},
		IsDeleted: true,
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
				var resp Vomit
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindPayload, resp.Kind)
				require.Equal(t, ReqInvalidArguments, resp.Reason)
				require.Equal(t, http.StatusBadRequest, resp.Status)
				require.Equal(t, `invalid user id: "12s"`, resp.ErrMessage)
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
				var resp Vomit
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindPayload, resp.Kind)
				require.Equal(t, ReqInvalidArguments, resp.Reason)
				require.Equal(t, http.StatusBadRequest, resp.Status)
				require.Equal(t, `invalid user id: "-1"`, resp.ErrMessage)
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
				require.Equal(t, http.StatusNotFound, resp.Status)
				require.Equal(t, db.KindNotFound.String(), resp.Reason)
				require.Equal(t, fmt.Sprintf("user with id %d not found", user.ID), resp.Error)
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
				require.Equal(t, http.StatusInternalServerError, resp.Status)
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
				require.Equal(t, http.StatusNotFound, resp.Status)
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
				require.Equal(t, user.DisplayName, resp.DisplayName)
				require.Equal(t, user.CreatedAt, resp.CreatedAt)
				require.NotNil(t, resp.ProfileImageURL)
				require.Equal(t, user.ProfileImgUrl.String, *resp.ProfileImageURL)

				var raw map[string]any
				err = json.Unmarshal(recorder.Body.Bytes(), &raw)
				require.NoError(t, err)
				require.NotContains(t, raw, "username")
				require.NotContains(t, raw, "email")
				require.NotContains(t, raw, "is_deleted")
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
			require.Equal(t, contentJSON, recorder.Header().Get("Content-Type"))
			tc.checkResponse(t, recorder)
		})
	}
}
