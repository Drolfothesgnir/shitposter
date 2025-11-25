package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/Drolfothesgnir/shitposter/db/mock"
	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/jackc/pgx/v5"
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
			name: "NotFoundIdempotent",
			url:  "/posts/1/comments/10",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteCommentTx(gomock.Any(), db.DeleteCommentTxParams{
						CommentID: commentID,
						UserID:    userID,
						PostID:    postID,
					}).
					Times(1).
					Return(db.DeleteCommentTxResult{}, db.ErrEntityNotFound)
			},
			setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
				setAuthorizationHeader(t, maker, authorizationTypeBearer, userID, time.Minute, req)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNoContent, rec.Code)
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
					Return(db.DeleteCommentTxResult{}, db.ErrEntityDoesNotBelongToUser)
			},
			setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
				setAuthorizationHeader(t, maker, authorizationTypeBearer, userID, time.Minute, req)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, rec.Code)

				var resp ErrorResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)

				require.Equal(t, ErrInvalidCommentID.Error(), resp.Error)
				require.Len(t, resp.Fields, 1)

				field := resp.Fields[0]
				require.Equal(t, "user_id", field.FieldName)
				require.Contains(t, field.ErrorMessage,
					"Comment with ID [10] does not belong to the user with ID [1]")
			},
		},
		{
			name: "InvalidPostID",
			url:  "/posts/1/comments/10",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteCommentTx(gomock.Any(), db.DeleteCommentTxParams{
						CommentID: commentID,
						UserID:    userID,
						PostID:    postID,
					}).
					Times(1).
					Return(db.DeleteCommentTxResult{}, db.ErrInvalidPostID)
			},
			setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
				setAuthorizationHeader(t, maker, authorizationTypeBearer, userID, time.Minute, req)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, rec.Code)

				var resp ErrorResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)

				require.Equal(t, ErrInvalidPostID.Error(), resp.Error)
				require.Len(t, resp.Fields, 1)

				field := resp.Fields[0]
				require.Equal(t, "post_id", field.FieldName)
				require.Contains(t, field.ErrorMessage,
					"Comment with ID [10] does not belong to the post with ID [1]")
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
					Return(db.DeleteCommentTxResult{}, pgx.ErrTxClosed)
			},
			setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
				setAuthorizationHeader(t, maker, authorizationTypeBearer, userID, time.Minute, req)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)

				var resp ErrorResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)

				require.Equal(t, ErrCannotDelete.Error(), resp.Error)
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
