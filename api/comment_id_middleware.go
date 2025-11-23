package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// This middleware checks the mandatory comment ID parameter in the URL.
//
// I chose to use middleware instead of Gin's URI binding because it is
// harder to produce a human-readable error message with the binder than
// with manual validation. It also makes handlers cleaner.
func (s *Service) commentIDMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		desc := getCommentIDDescriptor(ctx)
		if !desc.available() {
			field := ErrorField{"comment_id", fmt.Sprintf("comment id [%s] is invalid", desc.rawValue)}
			ctx.AbortWithStatusJSON(
				http.StatusBadRequest,
				NewErrorResponse(ErrInvalidCommentID, field),
			)
			return
		}

		ctx.Set(ctxCommentIDKey, desc.parsedValue)
		ctx.Next()
	}
}

// Helper function to get the comment ID after middleware check.
func extractCommentIDFromCtx(ctx *gin.Context) int64 {
	return ctx.MustGet(ctxCommentIDKey).(int64)
}
