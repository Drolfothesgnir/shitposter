package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/Drolfothesgnir/shitposter/db/mock"
	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetComments(t *testing.T) {
	var postID int64 = 1

	testCases := []struct {
		name          string
		query         string
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:  "InvalidOrder",
			query: "order=inv",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().QueryComments(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				var resp PayloadError
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, KindPayload, resp.Kind)
				require.Len(t, resp.Issues, 1)
				require.Equal(t, "order", resp.Issues[0].FieldName)
				require.Equal(t, "comment_order", resp.Issues[0].Tag)
			},
		},
		{
			name:  "OKNoRows",
			query: fmt.Sprintf("order=%s&root_offset=10&n_roots=10", db.CommentOrderPopular),
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CommentQuery{
					PostID: postID,
					Limit:  10,
					Offset: 10,
					Order:  db.CommentOrderPopular,
				}
				store.EXPECT().QueryComments(gomock.Any(), arg).Times(1).Return([]db.CommentsWithAuthor{}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var res GetCommentsResponse
				err := json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				require.Len(t, res.Comments, 0)
			},
		},
		{
			name:  "QueryCommentsErr",
			query: fmt.Sprintf("order=%s&root_offset=45&n_roots=15", db.CommentOrderNewest),
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CommentQuery{
					PostID: postID,
					Limit:  15,
					Offset: 45,
					Order:  db.CommentOrderNewest,
				}
				store.EXPECT().QueryComments(gomock.Any(), arg).Times(1).Return(
					[]db.CommentsWithAuthor{},
					&db.OpError{
						Op:     "query-comments",
						Kind:   db.KindInternal,
						Entity: "comment",
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
				require.Equal(t, "internal", resp.Reason)
				require.Equal(t, "an internal error occurred", resp.Error)
			},
		},
		{
			name:  "PrepareCommentsBadDepthJump",
			query: fmt.Sprintf("order=%s&root_offset=45&n_roots=15", db.CommentOrderOldest),
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CommentQuery{
					PostID: postID,
					Limit:  15,
					Offset: 45,
					Order:  db.CommentOrderOldest,
				}

				var parentID int64 = 1
				comments := []db.CommentsWithAuthor{
					makeComment(1, 0, nil),       // root
					makeComment(2, 2, &parentID), // depth jump: 2 > len(stack)=1
				}

				store.EXPECT().QueryComments(gomock.Any(), arg).Times(1).Return(comments, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:  "OK",
			query: fmt.Sprintf("order=%s&root_offset=45&n_roots=15", db.CommentOrderOldest),
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CommentQuery{
					PostID: postID,
					Limit:  15,
					Offset: 45,
					Order:  db.CommentOrderOldest,
				}

				comments := make([]db.CommentsWithAuthor, 5)
				for i := range 5 {
					comments[i] = makeComment(int64(i+1), 0, nil)
				}

				store.EXPECT().QueryComments(gomock.Any(), arg).Times(1).Return(comments, nil)
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

			tc.buildStubs(store)

			service := newTestService(t, store, nil, nil, nil)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/posts/%d/comments?%s", postID, tc.query)

			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			service.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

// helper: minimal comment row
func makeComment(id int64, depth int32, parentID *int64) db.CommentsWithAuthor {
	c := db.CommentsWithAuthor{
		ID:    id,
		Depth: depth,
	}
	if parentID != nil {
		c.ParentID = pgtype.Int8{Int64: *parentID, Valid: true}
	} else {
		c.ParentID = pgtype.Int8{Valid: false}
	}
	return c
}
