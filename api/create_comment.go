package api

import (
	"fmt"
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateCommentRequest struct {
	Body string `json:"body" binding:"required,max=500"`
}

func (s *Service) createComment(ctx *gin.Context) {
	// get token after auth middleware use
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// get post id after post id check middleware
	postID := ctx.MustGet(ctxPostIDKey).(int64)

	var req CreateCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(
			http.StatusBadRequest,
			NewErrorResponse(ErrInvalidParams, ExtractErrorFields(err)...))
		return
	}

	// extracting comment id to check if comment is a reply
	// i.e. comment_id from /posts/:post_id/comments/:comment_id is available
	desc := getCommentIDDescriptor(ctx)

	// if the comment_id provided but not valid abort with 400
	if !desc.valid && desc.provided {
		errField := ErrorField{"comment_id", fmt.Sprintf("Cannot reply to the comment with id: %s", desc.rawValue)}
		ctx.JSON(
			http.StatusBadRequest,
			NewErrorResponse(ErrInvalidParentCommentId, errField),
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
		switch err {
		case db.ErrInvalidPostID:
			errField := ErrorField{"post_id", fmt.Sprintf("Invalid post id: %d", postID)}
			ctx.JSON(
				http.StatusBadRequest,
				NewErrorResponse(ErrInvalidPostID, errField),
			)
			return
		case db.ErrParentCommentNotFound, db.ErrParentCommentPostIDMismatch:
			errField := ErrorField{"comment_id", fmt.Sprintf("Cannot reply to the comment with id: %s", desc.rawValue)}
			ctx.JSON(
				http.StatusBadRequest,
				NewErrorResponse(ErrInvalidParentCommentId, errField),
			)
			return
		case db.ErrParentCommentDeleted:
			errField := ErrorField{
				"comment_id",
				fmt.Sprintf(
					"Comment with id [%s] is deleted. Can't reply to a deleted comment",
					desc.rawValue,
				)}
			ctx.JSON(
				http.StatusBadRequest,
				NewErrorResponse(ErrInvalidParentCommentId, errField),
			)
			return
		default:
			ctx.JSON(http.StatusInternalServerError, err)
			return
		}
	}

	ctx.JSON(http.StatusOK, comment)
}
