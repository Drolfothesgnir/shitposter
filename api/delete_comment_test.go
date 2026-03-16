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
	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDeleteComment(t *testing.T) {
	userID := int64(1)
	postID := int64(1)
	commentID := int64(10)

	testCases := []struct {
		name          string
		url           string
		buildStubs    func(store *mockdb.MockStore)
		setupAuth     func(t *testing.T, req *http.Request, maker token.Maker)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "OKDeleted",
			url:  "/posts/1/comments/10",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteCommentTx(gomock.Any(), db.DeleteCommentTxParams{
						CommentID: commentID,
						UserID:    userID,
						PostID:    postID,
					}).
					Times(1).
					Return(db.DeleteCommentTxResult{}, nil)
			},
			setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
				setAuthorizationHeader(t, maker, authorizationTypeBearer, userID, time.Minute, req)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNoContent, rec.Code)
			},
		},
		{
			name: "NotFound",
			url:  "/posts/1/comments/10",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteCommentTx(gomock.Any(), db.DeleteCommentTxParams{
						CommentID: commentID,
						UserID:    userID,
						PostID:    postID,
					}).
					Times(1).
					Return(db.DeleteCommentTxResult{}, &db.OpError{
						Op:       "delete-comment",
						Kind:     db.KindNotFound,
						Entity:   "comment",
						EntityID: fmt.Sprint(commentID),
						Err:      fmt.Errorf("comment with id %d not found", commentID),
					})
			},
			setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
				setAuthorizationHeader(t, maker, authorizationTypeBearer, userID, time.Minute, req)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
				var resp ResourceError
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, "not_found", resp.Reason)
			},
		},
		{
			name: "ForbiddenWrongUser",
			url:  "/posts/1/comments/10",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteCommentTx(gomock.Any(), db.DeleteCommentTxParams{
						CommentID: commentID,
						UserID:    userID,
						PostID:    postID,
					}).
					Times(1).
					Return(db.DeleteCommentTxResult{}, &db.OpError{
						Op:       "delete-comment",
						Kind:     db.KindPermission,
						Entity:   "comment",
						EntityID: fmt.Sprint(commentID),
						UserID:   fmt.Sprint(userID),
						Err:      fmt.Errorf("comment with id %d does not belong to user with id %d", commentID, userID),
					})
			},
			setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
				setAuthorizationHeader(t, maker, authorizationTypeBearer, userID, time.Minute, req)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, rec.Code)
				var resp ResourceError
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, "permission", resp.Reason)
				require.Contains(t, resp.Error, "does not belong to user")
			},
		},
		{
			name: "PostMismatch",
			url:  "/posts/1/comments/10",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteCommentTx(gomock.Any(), db.DeleteCommentTxParams{
						CommentID: commentID,
						UserID:    userID,
						PostID:    postID,
					}).
					Times(1).
					Return(db.DeleteCommentTxResult{}, &db.OpError{
						Op:              "delete-comment",
						Kind:            db.KindRelation,
						Entity:          "comment",
						EntityID:        fmt.Sprint(commentID),
						RelatedEntity:   "post",
						RelatedEntityID: fmt.Sprint(postID),
						FailingField:    "post_id",
						Err:             fmt.Errorf("comment with id %d does not belong to post with id %d", commentID, postID),
					})
			},
			setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
				setAuthorizationHeader(t, maker, authorizationTypeBearer, userID, time.Minute, req)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
				var resp ResourceError
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, "relation", resp.Reason)
				require.Contains(t, resp.Error, "does not belong to post")
			},
		},
		{
			name: "InternalError",
			url:  "/posts/1/comments/10",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteCommentTx(gomock.Any(), db.DeleteCommentTxParams{
						CommentID: commentID,
						UserID:    userID,
						PostID:    postID,
					}).
					Times(1).
					Return(db.DeleteCommentTxResult{}, &db.OpError{
						Op:     "delete-comment",
						Kind:   db.KindInternal,
						Entity: "comment",
						Err:    fmt.Errorf("tx closed"),
					})
			},
			setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
				setAuthorizationHeader(t, maker, authorizationTypeBearer, userID, time.Minute, req)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
				var resp ResourceError
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, "internal", resp.Reason)
				require.Equal(t, "an internal error occurred", resp.Error)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tokenMaker, err := token.NewJWTMaker(testConfig.TokenSymmetricKey)
			require.NoError(t, err)

			tc.buildStubs(store)

			service := newTestService(t, store, tokenMaker, nil, nil)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(http.MethodDelete, tc.url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, req, tokenMaker)

			service.router.ServeHTTP(rec, req)
			tc.checkResponse(t, rec)
		})
	}
}
