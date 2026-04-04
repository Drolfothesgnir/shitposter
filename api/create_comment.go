package api

import (
	"fmt"
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateCommentRequest struct {
	Body string `json:"body"`
}

func (r CreateCommentRequest) Validate() *Vomit {
	issues := make([]Issue, 0)
	validate(&issues, r.Body, "body", strRequired, strMax(500))
	return barf(issues)
}

func (s *Service) createComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authPayload := getAuthPayload(ctx)

	postID, vErr := extractPostID(r)
	if vErr != nil {
		abortWithError(w, vErr)
		return
	}

	var req CreateCommentRequest
	if vErr := ingestJSONBody(w, r, &req); vErr != nil {
		abortWithError(w, vErr)
		return
	}

	// extracting comment id to check if comment is a reply
	// i.e. comment_id from /posts/:post_id/comments/:comment_id is available
	desc := getCommentIDDescriptor(r)

	// if the comment_id provided but not valid abort with 400
	if !desc.valid && desc.provided {
		vErr := puke(
			ReqInvalidArguments,
			http.StatusBadRequest,
			fmt.Sprintf("invalid comment id: %s", desc.rawValue),
			nil,
		)
		abortWithError(w, vErr)
		return
	}

	// otherwise assume the comment is a reply
	arg := db.InsertCommentTxParams{
		UserID:   authPayload.UserID,
		PostID:   postID,
		Body:     req.Body,
		ParentID: pgtype.Int8{Int64: desc.parsedValue, Valid: desc.valid},
	}

	comment, err := s.store.InsertCommentTx(ctx, arg)
	if err != nil {
		opErr := newResourceError(err)
		abortWithError(w, opErr)
		return
	}

	respondWithJSON(w, http.StatusOK, comment)
}
