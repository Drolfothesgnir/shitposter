package api

import (
	"errors"
	"fmt"
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

	if err == nil || errors.Is(err, db.ErrEntityNotFound) {
		// idempotent delete: 204 even if comment didn’t exist
		ctx.Status(http.StatusNoContent)
		return
	}

	switch {
	case errors.Is(err, db.ErrEntityDoesNotBelongToUser):
		errField := ErrorField{
			FieldName: "user_id",
			ErrorMessage: fmt.Sprintf(
				"Comment with ID [%d] does not belong to the user with ID [%d]",
				commentID, authPayload.UserID,
			),
		}
		ctx.JSON(
			http.StatusForbidden,
			NewErrorResponse(ErrInvalidCommentID, errField),
		)
		return

	case errors.Is(err, db.ErrInvalidPostID):
		errField := ErrorField{
			FieldName: "post_id",
			ErrorMessage: fmt.Sprintf(
				"Comment with ID [%d] does not belong to the post with ID [%d]",
				commentID, postID,
			),
		}
		ctx.JSON(
			http.StatusConflict,
			NewErrorResponse(ErrInvalidPostID, errField),
		)
		return

	default:
		// in case of any db error
		ctx.JSON(
			http.StatusInternalServerError,
			NewErrorResponse(ErrCannotDelete),
		)
		return
	}

}
