package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// helper to create router with middleware wired same way as in SetupRouter
func setupPostIDTestRouter(s *Service, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()

	// simulate: publicPostGroup := router.Group("/posts").Use(service.postIDMiddleware())
	group := r.Group("/posts").Use(s.postIDMiddleware())
	group.GET("/:post_id/comments", handler)

	return r
}

// 1. Valid post_id: middleware should set ctx value and let handler run
func TestPostIDMiddleware_ValidPostID(t *testing.T) {
	s := &Service{} // we don't need any fields for this middleware

	called := false

	router := setupPostIDTestRouter(s, func(ctx *gin.Context) {
		called = true

		v, exists := ctx.Get(postIDKey)
		require.True(t, exists, "post_id should be set in context by middleware")

		postID, ok := v.(int64)
		require.True(t, ok, "post_id in context should be int64")
		require.Equal(t, int64(123), postID)

		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/posts/123/comments", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.True(t, called, "handler should be called for valid post_id")
	require.Equal(t, http.StatusOK, resp.Code)
}

// 2. Invalid post_id: middleware should abort with 400 and NOT call handler
func TestPostIDMiddleware_InvalidPostID(t *testing.T) {
	s := &Service{}

	called := false

	router := setupPostIDTestRouter(s, func(ctx *gin.Context) {
		called = true // should NOT be reached
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/posts/abc/comments", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.False(t, called, "handler should NOT be called for invalid post_id")
	require.Equal(t, http.StatusBadRequest, resp.Code)

	// Optional: check that response contains our error message stub
	body := resp.Body.String()
	require.Contains(t, body, "Invalid post id", "response body should contain error message")
}
