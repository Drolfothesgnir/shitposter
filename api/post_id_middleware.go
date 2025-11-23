package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// This middleware checks the mandatory post ID parameter in the URL.
//
// I chose to use middleware instead of Gin's URI binding because it is
// harder to produce a human-readable error message with the binder than
// with manual validation. It also makes handlers cleaner.
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

		ctx.Set(ctxPostIDKey, postID)
		ctx.Next()
	}
}

// Helper function to get the post ID after middleware check.
func extractPostIDFromCtx(ctx *gin.Context) int64 {
	return ctx.MustGet(ctxPostIDKey).(int64)
}
