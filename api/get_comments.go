package api

import (
	"errors"
	"net/http"
	"net/url"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
)

type GetCommentsRequest struct {
	RootOffset int32           `json:"root_offset"`
	NRoots     int32           `json:"n_roots"`
	Order      db.CommentOrder `json:"order"`
}

func (r *GetCommentsRequest) ExtractQueryParams(m url.Values) *Vomit {
	issues := make([]Issue, 0, 3)

	// root_offset
	extractOptionalParam(&issues, m, "root_offset", &r.RootOffset, parseSingle(parseInt32), numMin(int32(0)))

	// n_roots
	extractOptionalParam(&issues, m, "n_roots", &r.NRoots, parseSingle(parseInt32), numMin(int32(1)), numMax(int32(100)))

	// order
	extractOptionalParam(&issues, m, "order", &r.Order, parseSingle(parseOrder), valCommentOrder)

	return barf(issues)
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

func parseOrder(s string) (db.CommentOrder, error) {
	o, err := parseString(s)
	return db.CommentOrder(o), err
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

	if vErr := req.ExtractQueryParams(r.URL.Query()); vErr != nil {
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
