package api

import (
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/gin-gonic/gin"
)

func (s *Service) deleteComment(ctx *gin.Context) {
	authPayload := extractAuthPayloadFromCtx(ctx)
	postID := extractPostIDFromCtx(ctx)
	commentID := extractCommentIDFromCtx(ctx)

	_, err := s.store.DeleteCommentTx(ctx, db.DeleteCommentTxParams{
		CommentID: commentID,
		UserID:    authPayload.UserID,
		PostID:    postID,
	})

	if err != nil {
		opErr := newResourceError(err)
		ctx.JSON(opErr.StatusCode(), opErr)
		return
	}

	ctx.Status(http.StatusNoContent)
}
