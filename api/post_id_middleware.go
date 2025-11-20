package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const postIDKey = "provided_post_id"

func (s *Service) postIDMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// getting mandatory post id form the request, abort with 400 on error
		postIDRaw := ctx.Param("post_id")

		postID, err := strconv.ParseInt(postIDRaw, 10, 64)
		if err != nil {
			errField := ErrorField{"post_id", fmt.Sprintf("Invalid post id: %s", postIDRaw)}
			ctx.AbortWithStatusJSON(
				http.StatusBadRequest,
				NewErrorResponse(ErrInvalidPostID, errField),
			)
			return
		}

		ctx.Set(postIDKey, postID)
		ctx.Next()
	}
}
