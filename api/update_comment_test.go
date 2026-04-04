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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUpdateComment(t *testing.T) {

	arg := db.UpdateCommentParams{
		CommentID: 1,
		UserID:    1,
		PostID:    1,
		Body:      "test",
	}

	testCases := []struct {
		name          string
		body          reqBody
		buildStubs    func(store *mockdb.MockStore)
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "EmptyBody",
			body: reqBody{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateComment(gomock.Any(), gomock.Any()).Times(0)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, 1, time.Minute, request)
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
				require.Equal(t, "required", resp.Issues[0].Tag)
			},
		},
		{
			name: "TargetNotFound",
			body: reqBody{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateComment(gomock.Any(), arg).Times(1).Return(
					db.UpdateCommentResult{},
					&db.OpError{
						Op:       "update-comment",
						Kind:     db.KindNotFound,
						Entity:   "comment",
						EntityID: "1",
						Err:      fmt.Errorf("comment with id 1 not found"),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, 1, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, http.StatusNotFound, resp.Status)
				require.Equal(t, "not_found", resp.Reason)
				require.Equal(t, "comment with id 1 not found", resp.Error)
			},
		},
		{
			name: "UpdateCommentDBErr",
			body: reqBody{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateComment(gomock.Any(), arg).Times(1).Return(
					db.UpdateCommentResult{},
					&db.OpError{
						Op:     "update-comment",
						Kind:   db.KindInternal,
						Entity: "comment",
						Err:    fmt.Errorf("tx closed"),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, 1, time.Minute, request)
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
			name: "TargetDeleted",
			body: reqBody{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateComment(gomock.Any(), arg).Times(1).Return(
					db.UpdateCommentResult{},
					&db.OpError{
						Op:       "update-comment",
						Kind:     db.KindDeleted,
						Entity:   "comment",
						EntityID: "1",
						Err:      fmt.Errorf("comment with id 1 is deleted and cannot be updated"),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, 1, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusGone, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, http.StatusGone, resp.Status)
				require.Equal(t, "deleted", resp.Reason)
				require.Equal(t, "comment with id 1 is deleted and cannot be updated", resp.Error)
			},
		},
		{
			name: "UserIDMismatch",
			body: reqBody{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateComment(gomock.Any(), arg).Times(1).Return(
					db.UpdateCommentResult{},
					&db.OpError{
						Op:       "update-comment",
						Kind:     db.KindPermission,
						Entity:   "comment",
						EntityID: "1",
						UserID:   "1",
						Err:      fmt.Errorf("comment with id 1 does not belong to user with id 1"),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, 1, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, http.StatusForbidden, resp.Status)
				require.Equal(t, "permission", resp.Reason)
				require.Equal(t, "comment with id 1 does not belong to user with id 1", resp.Error)
			},
		},
		{
			name: "PostIDMismatch",
			body: reqBody{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateComment(gomock.Any(), arg).Times(1).Return(
					db.UpdateCommentResult{},
					&db.OpError{
						Op:       "update-comment",
						Kind:     db.KindRelation,
						Entity:   "comment",
						EntityID: "1",
						Err:      fmt.Errorf("comment with id 1 does not belong to post with id 1"),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, 1, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, http.StatusBadRequest, resp.Status)
				require.Equal(t, "relation", resp.Reason)
				require.Equal(t, "comment with id 1 does not belong to post with id 1", resp.Error)
			},
		},
		{
			name: "OK",
			body: reqBody{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateComment(gomock.Any(), arg).Times(1).Return(
					db.UpdateCommentResult{
						ID:   1,
						Body: "test",
					},
					nil,
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, 1, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var resp db.UpdateCommentResult
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, int64(1), resp.ID)
				require.Equal(t, "test", resp.Body)
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

			url := "/posts/1/comments/1"
			request, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, tokenMaker)

			service.router.ServeHTTP(recorder, request)
			require.Equal(t, contentJSON, recorder.Header().Get("Content-Type"))
			tc.checkResponse(t, recorder)
		})
	}
}
