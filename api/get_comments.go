package api

import (
	"errors"
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
)

type GetCommentsRequest struct {
	RootOffset int32           `json:"root_offset"`
	NRoots     int32           `json:"n_roots"`
	Order      db.CommentOrder `json:"order"`
}

// TODO: is word "order" appropriate here? shouldn't it be "sort order" or "sort" or something
// there and in the db?
func valCommentOrder(v db.CommentOrder, fieldname string, issues *[]Issue) bool {
	err := db.ValidateCommentOrderMethod(v)
	if err != nil {
		var opErr *db.OpError
		var msg string
		if errors.As(err, &opErr) {
			msg = opErr.Error()
		} else {
			msg = "invalid comments order method"
		}
		*issues = append(*issues, Issue{
			FieldName: fieldname,
			Tag:       "comment_order",
			Message:   msg,
		})
	}

	return true
}

func (r GetCommentsRequest) Validate() *Vomit {
	issues := make([]Issue, 0, 3)
	validate(&issues, r.RootOffset, "root_offset", numMin(int32(0)))
	validate(&issues, r.NRoots, "n_roots", numMin(int32(1)), numMax(int32(100)))
	validate(&issues, r.Order, "order", valCommentOrder)
	return barf(issues)
}

type GetCommentsResponse struct {
	Comments []*CommentNode `json:"comments"`
}

func (s *Service) getComments(w http.ResponseWriter, r *http.Request) {
	postID, vErr := extractPostID(r)
	if vErr != nil {
		abortWithError(w, vErr)
		return
	}

	// pre-filled with default values
	req := GetCommentsRequest{
		RootOffset: 0,
		NRoots:     10,
		Order:      db.CommentOrderPopular,
	}

	if vErr := ingestJSONBody(w, r, &req); vErr != nil {
		respondWithJSON(w, vErr.Status, vErr)
		return
	}

	if vErr := req.Validate(); vErr != nil {
		respondWithJSON(w, vErr.Status, vErr)
		return
	}

	query := db.CommentQuery{
		PostID: postID,
		Order:  req.Order,
		Limit:  req.NRoots,
		Offset: req.RootOffset,
	}

	ctx := r.Context()

	comments, err := s.store.QueryComments(ctx, query)

	if err != nil {
		opErr := newResourceError(err)
		abortWithError(w, opErr)
		return
	}

	tree, err := PrepareCommentTree(comments, int(req.NRoots))

	// in case the tree cannot be formed, then there should be some data corruption in the db
	// abort with 500
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, internalResourceError())
		return
	}

	respondWithJSON(w, http.StatusOK, GetCommentsResponse{tree})
}
