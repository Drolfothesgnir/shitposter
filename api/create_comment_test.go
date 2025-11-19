package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mockdb "github.com/Drolfothesgnir/shitposter/db/mock"
	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
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
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, ErrInvalidParams.Error(), res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, res.Fields[0].FieldName, "body")
				require.Equal(t, res.Fields[0].ErrorMessage, getBindingErrorMessage("required"))
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
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, ErrInvalidParams.Error(), res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, res.Fields[0].FieldName, "body")
				require.Equal(t, res.Fields[0].ErrorMessage, getBindingErrorMessage("max"))
			},
		},
		{
			name: "InvalidPostId",
			url:  "/posts/invalid_id/comments",
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
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, ErrInvalidPostID.Error(), res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, res.Fields[0].FieldName, "post_id")
				require.Equal(t, res.Fields[0].ErrorMessage, "Invalid post id: invalid_id")
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
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, ErrInvalidParentCommentId.Error(), res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, res.Fields[0].FieldName, "comment_id")
				require.Equal(t, res.Fields[0].ErrorMessage, "Cannot reply to the comment with id: inv_par_id")
			},
		},
		{
			name: "InvalidPostID",
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
				store.EXPECT().InsertCommentTx(gomock.Any(), arg).Times(1).Return(db.Comment{}, db.ErrInvalidPostID)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, ErrInvalidPostID.Error(), res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, res.Fields[0].FieldName, "post_id")
				require.Equal(t, res.Fields[0].ErrorMessage, "Invalid post id: 1")
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
				store.EXPECT().InsertCommentTx(gomock.Any(), arg).Times(1).Return(db.Comment{}, db.ErrParentCommentNotFound)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, ErrInvalidParentCommentId.Error(), res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, res.Fields[0].FieldName, "comment_id")
				require.Equal(t, res.Fields[0].ErrorMessage, "Cannot reply to the comment with id: 1")
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
				store.EXPECT().InsertCommentTx(gomock.Any(), arg).Times(1).Return(db.Comment{}, db.ErrParentCommentPostIDMismatch)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, ErrInvalidParentCommentId.Error(), res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, res.Fields[0].FieldName, "comment_id")
				require.Equal(t, res.Fields[0].ErrorMessage, "Cannot reply to the comment with id: 1")
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
				store.EXPECT().InsertCommentTx(gomock.Any(), arg).Times(1).Return(db.Comment{}, db.ErrParentCommentDeleted)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, ErrInvalidParentCommentId.Error(), res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, res.Fields[0].FieldName, "comment_id")
				require.Equal(t, res.Fields[0].ErrorMessage, "Comment with id [1] is deleted. Can't reply to a deleted comment")
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
				store.EXPECT().InsertCommentTx(gomock.Any(), arg).Times(1).Return(db.Comment{}, pgx.ErrTxClosed)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, user.ID, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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
