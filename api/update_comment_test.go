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
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUpdateComment(t *testing.T) {

	arg := db.UpdateCommentParams{
		PCommentID: 1,
		PUserID:    1,
		PPostID:    1,
		PBody:      "test",
	}

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "EmptyBody",
			body: gin.H{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateComment(gomock.Any(), gomock.Any()).Times(0)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, 1, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, ErrInvalidParams.Error(), res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, "body", res.Fields[0].FieldName)
				require.Equal(t, getBindingErrorMessage("required", "", ""), res.Fields[0].ErrorMessage)
			},
		},
		{
			name: "TargetNotFound",
			body: gin.H{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateComment(gomock.Any(), arg).Times(1).Return(db.UpdateCommentRow{}, pgx.ErrNoRows)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, 1, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, ErrInvalidCommentID.Error(), res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, "comment_id", res.Fields[0].FieldName)
				require.Equal(t, "Comment with ID [1] does not exist", res.Fields[0].ErrorMessage)
			},
		},
		{
			name: "UpdateCommentDBErr",
			body: gin.H{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateComment(gomock.Any(), arg).Times(1).Return(db.UpdateCommentRow{}, pgx.ErrTxClosed)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, 1, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "TargetDeleted",
			body: gin.H{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				res := db.UpdateCommentRow{
					IsDeleted: true,
				}
				store.EXPECT().UpdateComment(gomock.Any(), arg).Times(1).Return(res, nil)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, 1, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusGone, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, ErrCommentDeleted.Error(), res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, "comment_id", res.Fields[0].FieldName)
				require.Equal(t, "Comment with ID [1] is deleted and cannot be updated", res.Fields[0].ErrorMessage)
			},
		},
		{
			name: "UserIDMismatch",
			body: gin.H{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				res := db.UpdateCommentRow{
					UserID: 2,
				}
				store.EXPECT().UpdateComment(gomock.Any(), arg).Times(1).Return(res, nil)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, 1, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, ErrCannotUpdate.Error(), res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, "user_id", res.Fields[0].FieldName)
				require.Equal(t, "This comment does not belong to the authenticated user", res.Fields[0].ErrorMessage)
			},
		},
		{
			name: "PostIDMismatch",
			body: gin.H{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				res := db.UpdateCommentRow{
					UserID: 1,
					PostID: 2,
				}
				store.EXPECT().UpdateComment(gomock.Any(), arg).Times(1).Return(res, nil)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, 1, time.Minute, request)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
				res, err := extractErrorFromBuffer(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, ErrInvalidPostID.Error(), res.Error)
				require.Len(t, res.Fields, 1)
				require.Equal(t, "post_id", res.Fields[0].FieldName)
				require.Equal(t, "Comment with ID [1] does not belong to post with ID [1]", res.Fields[0].ErrorMessage)
			},
		},
		{
			name: "OK",
			body: gin.H{
				"body": "test",
			},
			buildStubs: func(store *mockdb.MockStore) {
				res := db.UpdateCommentRow{
					Updated: true,
				}
				store.EXPECT().UpdateComment(gomock.Any(), arg).Times(1).Return(res, nil)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				setAuthorizationHeader(t, tokenMaker, authorizationTypeBearer, 1, time.Minute, request)
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

			url := "/posts/1/comments/1"
			request, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, tokenMaker)

			service.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
