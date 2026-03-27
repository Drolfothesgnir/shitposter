package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mockdb "github.com/Drolfothesgnir/shitposter/db/mock"
	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateComment(t *testing.T) {
	user := db.User{
		ID: 1,
	}

	testCases := []struct {
		name          string
		url           string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "EmptyBody",
			url:  "/posts/1/comments",
			body: gin.H{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().InsertCommentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				var resp PayloadError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindPayload, resp.Kind)
				require.Len(t, resp.Issues, 1)
				require.Equal(t, "body", resp.Issues[0].FieldName)
				require.Equal(t, "required", resp.Issues[0].Reason)
			},
		},
		{
			name: "BodyTooLong",
			url:  "/posts/1/comments",
			body: gin.H{
				"body": strings.Repeat("too long", 100),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().InsertCommentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				var resp PayloadError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindPayload, resp.Kind)
				require.Len(t, resp.Issues, 1)
				require.Equal(t, "body", resp.Issues[0].FieldName)
				require.Equal(t, "max", resp.Issues[0].Reason)
			},
		},
		{
			name: "InvalidParentId",
			url:  "/posts/1/comments/inv_par_id",
			body: gin.H{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().InsertCommentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				var resp PayloadError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindPayload, resp.Kind)
				require.Contains(t, resp.Error, "invalid comment id")
			},
		},
		{
			name: "PostNotFound",
			url:  "/posts/1/comments",
			body: gin.H{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.InsertCommentTxParams{
					UserID: user.ID,
					PostID: 1,
					Body:   "test",
				}
				store.EXPECT().InsertCommentTx(gomock.Any(), arg).Times(1).Return(
					db.Comment{},
					&db.OpError{
						Op:              "insert-comment",
						Kind:            db.KindRelation,
						Entity:          "comment",
						RelatedEntity:   "post",
						RelatedEntityID: "1",
						Err:             fmt.Errorf("attempt to create comment for a non-existent post with id [1]"),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, "relation", resp.Reason)
				require.Contains(t, resp.Error, "post")
			},
		},
		{
			name: "MissingParentComment",
			url:  "/posts/1/comments/1",
			body: gin.H{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.InsertCommentTxParams{
					UserID:   user.ID,
					PostID:   1,
					Body:     "test",
					ParentID: pgtype.Int8{Int64: 1, Valid: true},
				}
				store.EXPECT().InsertCommentTx(gomock.Any(), arg).Times(1).Return(
					db.Comment{},
					&db.OpError{
						Op:            "insert-comment",
						Kind:          db.KindNotFound,
						Entity:        "comment",
						RelatedEntity: "comment",
						Err:           fmt.Errorf("cannot reply to the comment with id [1]: the comment doesn't exist"),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, "not_found", resp.Reason)
			},
		},
		{
			name: "ParentCommentPostIDMismatch",
			url:  "/posts/1/comments/1",
			body: gin.H{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.InsertCommentTxParams{
					UserID:   user.ID,
					PostID:   1,
					Body:     "test",
					ParentID: pgtype.Int8{Int64: 1, Valid: true},
				}
				store.EXPECT().InsertCommentTx(gomock.Any(), arg).Times(1).Return(
					db.Comment{},
					&db.OpError{
						Op:              "insert-comment",
						Kind:            db.KindRelation,
						Entity:          "comment",
						RelatedEntity:   "comment",
						RelatedEntityID: "1",
						FailingField:    "post_id",
						Err:             fmt.Errorf("cannot reply to comment with ID [1] for post with ID [1]: parent comment belongs to post with ID [2]"),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, "relation", resp.Reason)
			},
		},
		{
			name: "ParentCommentDeleted",
			url:  "/posts/1/comments/1",
			body: gin.H{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.InsertCommentTxParams{
					UserID:   user.ID,
					PostID:   1,
					Body:     "test",
					ParentID: pgtype.Int8{Int64: 1, Valid: true},
				}
				store.EXPECT().InsertCommentTx(gomock.Any(), arg).Times(1).Return(
					db.Comment{},
					&db.OpError{
						Op:       "insert-comment",
						Kind:     db.KindDeleted,
						Entity:   "comment",
						EntityID: "1",
						Err:      fmt.Errorf("cannot reply to the deleted comment with id [1]"),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusGone, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, "deleted", resp.Reason)
			},
		},
		{
			name: "InternalError",
			url:  "/posts/1/comments/1",
			body: gin.H{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.InsertCommentTxParams{
					UserID:   user.ID,
					PostID:   1,
					Body:     "test",
					ParentID: pgtype.Int8{Int64: 1, Valid: true},
				}
				store.EXPECT().InsertCommentTx(gomock.Any(), arg).Times(1).Return(
					db.Comment{},
					&db.OpError{
						Op:     "insert-comment",
						Kind:   db.KindInternal,
						Entity: "comment",
						Err:    fmt.Errorf("tx closed"),
					},
				)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				var resp ResourceError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindResource, resp.Kind)
				require.Equal(t, "internal", resp.Reason)
				require.Equal(t, "an internal error occurred", resp.Error)
			},
		},
		{
			name: "OKRoot",
			url:  "/posts/1/comments",
			body: gin.H{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.InsertCommentTxParams{
					UserID: user.ID,
					PostID: 1,
					Body:   "test",
				}

				comment := db.Comment{
					ID:     2,
					UserID: arg.UserID,
					PostID: 1,
				}

				store.EXPECT().InsertCommentTx(gomock.Any(), arg).Times(1).Return(comment, nil)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "OKReply",
			url:  "/posts/1/comments/1",
			body: gin.H{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.InsertCommentTxParams{
					UserID:   user.ID,
					PostID:   1,
					Body:     "test",
					ParentID: pgtype.Int8{Int64: 1, Valid: true},
				}

				comment := db.Comment{
					ID:       2,
					UserID:   arg.UserID,
					PostID:   1,
					ParentID: pgtype.Int8{Int64: 1, Valid: true},
					Depth:    1,
				}

				store.EXPECT().InsertCommentTx(gomock.Any(), arg).Times(1).Return(comment, nil)
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

			request, err := http.NewRequest(http.MethodPost, tc.url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, tokenMaker)

			service.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
