package api

import (
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
)

type UpdateCommentRequest struct {
	Body string `json:"body"`
}

func (r UpdateCommentRequest) Validate() *Vomit {
	issues := make([]Issue, 0)
	validate(&issues, r.Body, "body", strRequired, strMax(500))
	return barf(issues)
}

func (s *Service) updateComment(w http.ResponseWriter, r *http.Request) {
	var req UpdateCommentRequest
	if vErr := ingestJSONBody(w, r, &req); vErr != nil {
		respondWithJSON(w, vErr.Status, vErr)
		return
	}

	if vErr := req.Validate(); vErr != nil {
		respondWithJSON(w, vErr.Status, vErr)
		return
	}

	ctx := r.Context()

	authPayload := getAuthPayload(ctx)

	postID, vErr := extractPostID(r)
	if vErr != nil {
		abortWithError(w, vErr)
		return
	}

	commentID, vErr := extractCommentID(r)
	if vErr != nil {
		abortWithError(w, vErr)
		return
	}

	result, err := s.store.UpdateComment(ctx, db.UpdateCommentParams{
		CommentID: commentID,
		UserID:    authPayload.UserID,
		PostID:    postID,
		Body:      req.Body,
	})

	if err != nil {
		opErr := newResourceError(err)
		abortWithError(w, opErr)
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}
