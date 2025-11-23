package api

import (
	"fmt"
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type UpdateCommentRequest struct {
	Body string `json:"body" binding:"required,max=500"`
}

func (s *Service) updateComment(ctx *gin.Context) {
	var req UpdateCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(
			http.StatusBadRequest,
			NewErrorResponse(ErrInvalidParams, ExtractErrorFields(err)...),
		)
		return
	}

	authPayload := extractAuthPayloadFromCtx(ctx)
	postID := extractPostIDFromCtx(ctx)
	commentID := extractCommentIDFromCtx(ctx)

	result, err := s.store.UpdateComment(ctx, db.UpdateCommentParams{
		PCommentID: commentID,
		PUserID:    authPayload.UserID,
		PPostID:    postID,
		PBody:      req.Body,
	})

	// 1. The comment doesn't exist
	if err == pgx.ErrNoRows {
		errField := ErrorField{
			FieldName:    "comment_id",
			ErrorMessage: fmt.Sprintf("Comment with ID [%d] does not exist", commentID),
		}
		ctx.JSON(http.StatusNotFound, NewErrorResponse(ErrInvalidCommentID, errField))
		return
	}

	// 2. Any other db error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, NewErrorResponse(err))
		return
	}

	// 3. Update performed successfully
	if result.Updated {
		ctx.JSON(http.StatusOK, result)
		return
	}

	// 4. Update didn't happen, but the target comment exists
	//    Dealing with the issue in following order
	//    1) the comment is deleted
	//    2) the comment belongs to someone else
	//    3) the comment belongs to a different post

	// 4.1 The comment is deleted - most significant reason
	if result.IsDeleted {
		errField := ErrorField{
			FieldName:    "comment_id",
			ErrorMessage: fmt.Sprintf("Comment with ID [%d] is deleted and cannot be updated", commentID),
		}
		ctx.JSON(http.StatusGone, NewErrorResponse(ErrCommentDeleted, errField))
		return
	}

	// 4.2 The comment belongs to someone else
	if result.UserID != authPayload.UserID {
		errField := ErrorField{
			FieldName:    "user_id",
			ErrorMessage: "This comment does not belong to the authenticated user",
		}
		ctx.JSON(http.StatusForbidden, NewErrorResponse(ErrCannotUpdate, errField))
		return
	}

	// 4.3 The comment belongs to a different post
	if result.PostID != postID {
		errField := ErrorField{
			FieldName:    "post_id",
			ErrorMessage: fmt.Sprintf("Comment with ID [%d] does not belong to post with ID [%d]", commentID, postID),
		}
		ctx.JSON(http.StatusConflict, NewErrorResponse(ErrInvalidPostID, errField))
		return
	}

	// 4.4 Guarding fallback: in theory you can't fall into this case,
	//    but if the business logic above will change it's better to return error.
	ctx.JSON(http.StatusInternalServerError, NewErrorResponse(
		fmt.Errorf("comment update failed for unknown reason (comment_id=%d, user_id=%d, post_id=%d)", commentID, authPayload.UserID, postID),
	))
}
