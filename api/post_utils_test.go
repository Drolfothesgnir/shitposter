package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// 1. Valid post_id
func TestExtractPostID_ValidPostID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/posts/123/comments", nil)
	req.SetPathValue("post_id", "123")
	postID, vErr := extractPostID(req)
	require.Nil(t, vErr)
	require.Equal(t, int64(123), postID)
}

// 2. Invalid post_id
func TestPostIDMiddleware_InvalidPostID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/posts/abc/comments", nil)
	postID, err := extractPostID(req)

	require.Error(t, err)

	var vErr *Vomit
	require.ErrorAs(t, err, &vErr)

	require.Equal(t, ReqInvalidPostID, vErr.Reason)
	require.Equal(t, http.StatusBadRequest, vErr.Status)

	require.Equal(t, int64(-1), postID)
}
