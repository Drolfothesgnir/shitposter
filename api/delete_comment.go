package api

import (
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
)

func (s *Service) deleteComment(w http.ResponseWriter, r *http.Request) {
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

	_, err := s.store.DeleteCommentTx(ctx, db.DeleteCommentTxParams{
		CommentID: commentID,
		UserID:    authPayload.UserID,
		PostID:    postID,
	})

	if err != nil {
		opErr := newResourceError(err)
		abortWithError(w, opErr)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
