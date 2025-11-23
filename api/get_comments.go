package api

import (
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type GetCommentsQuery struct {
	RootOffset int32  `form:"root_offset" json:"root_offset" binding:"min=0"`
	NRoots     int32  `form:"n_roots" json:"n_roots" binding:"min=1,max=100"`
	Order      string `form:"order" json:"order" binding:"comment_order"`
}

type GetCommentsResponse struct {
	Comments []*CommentNode `json:"comments"`
}

func (s *Service) getComments(ctx *gin.Context) {
	postID := extractPostIDFromCtx(ctx)

	// pre-filled with default values
	req := GetCommentsQuery{
		RootOffset: 0,
		NRoots:     10,
		Order:      db.CommentOrderPopular,
	}

	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(
			http.StatusBadRequest,
			NewErrorResponse(ErrInvalidParams, ExtractErrorFields(err)...),
		)
		return
	}

	query := db.CommentQuery{
		PostID: postID,
		Order:  req.Order,
		Limit:  req.NRoots,
		Offset: req.RootOffset,
	}
	comments, err := s.store.QueryComments(ctx, query)

	// return empty response instead of 404 error when no rows found
	if err == pgx.ErrNoRows {
		ctx.JSON(http.StatusOK, GetCommentsResponse{})
		return
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, NewErrorResponse(err))
		return
	}

	tree, err := PrepareCommentTree(comments, int(req.NRoots))

	// in case the tree cannot be formed, then there should be some data corruption in the db
	// abort with 500
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, NewErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, GetCommentsResponse{tree})
}
