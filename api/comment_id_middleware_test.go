package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// helper to create router with middleware wired in the same way as SetupRouter
func setupCommentIDTestRouter(s *Service, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()

	// simulate:
	// privatePostCommentGroup := privatePostGroup.Use(service.commentIDMiddleware())
	group := r.Group("/posts/:post_id/comments").Use(s.commentIDMiddleware())
	group.GET("/:comment_id", handler)

	return r
}

// 1. Valid comment_id → middleware should set ctx value and call handler
func TestCommentIDMiddleware_ValidCommentID(t *testing.T) {
	s := &Service{} // middleware needs no service fields

	called := false

	router := setupCommentIDTestRouter(s, func(ctx *gin.Context) {
		called = true

		v, exists := ctx.Get(providedCommentID)
		require.True(t, exists, "comment_id should be set in context by middleware")

		commentID, ok := v.(int64)
		require.True(t, ok, "comment_id in context should be int64")
		require.Equal(t, int64(777), commentID)

		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/posts/123/comments/777", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.True(t, called, "handler should be called for valid comment_id")
	require.Equal(t, http.StatusOK, resp.Code)
}

// 2. Invalid comment_id → middleware must abort with 400 and not call handler
func TestCommentIDMiddleware_InvalidCommentID(t *testing.T) {
	s := &Service{}

	called := false

	router := setupCommentIDTestRouter(s, func(ctx *gin.Context) {
		called = true // should NOT run
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/posts/123/comments/abc", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.False(t, called, "handler should NOT be called for invalid comment_id")
	require.Equal(t, http.StatusBadRequest, resp.Code)

	body := resp.Body.String()
	require.Contains(t, body, "comment id", "response should contain error message")
}
