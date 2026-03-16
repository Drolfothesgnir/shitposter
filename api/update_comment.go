package api

import (
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/gin-gonic/gin"
)

type UpdateCommentRequest struct {
	Body string `json:"body" binding:"required,max=500"`
}

func (s *Service) updateComment(ctx *gin.Context) {
	var req UpdateCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, newPayloadError("invalid request parameters", err))
		return
	}

	authPayload := extractAuthPayloadFromCtx(ctx)
	postID := extractPostIDFromCtx(ctx)
	commentID := extractCommentIDFromCtx(ctx)

	result, err := s.store.UpdateComment(ctx, db.UpdateCommentParams{
		CommentID: commentID,
		UserID:    authPayload.UserID,
		PostID:    postID,
		Body:      req.Body,
	})

	if err != nil {
		opErr := newResourceError(err)
		ctx.JSON(opErr.StatusCode(), opErr)
		return
	}

	ctx.JSON(http.StatusOK, result)
}
