package api

import (
	"fmt"
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateCommentRequest struct {
	Body string `json:"body" binding:"required,max=500"`
}

func (s *Service) createComment(ctx *gin.Context) {
	authPayload := extractAuthPayloadFromCtx(ctx)

	postID := extractPostIDFromCtx(ctx)

	var req CreateCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(
			http.StatusBadRequest,
			newPayloadError("invalid request parameters", err))
		return
	}

	// extracting comment id to check if comment is a reply
	// i.e. comment_id from /posts/:post_id/comments/:comment_id is available
	desc := getCommentIDDescriptor(ctx)

	// if the comment_id provided but not valid abort with 400
	if !desc.valid && desc.provided {
		ctx.JSON(
			http.StatusBadRequest,
			newPayloadError(fmt.Sprintf("invalid comment id: %s", desc.rawValue), nil),
		)
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
		ctx.JSON(opErr.StatusCode(), opErr)
		return
	}

	ctx.JSON(http.StatusOK, comment)
}
